//
// SPDX-License-Identifier: BSD-3-Clause
//

// Package etagsim is a tiny Redfish HTTP service used by the opt-in
// integration test. It serves a ServiceRoot, a Chassis collection, and one
// Chassis member.
package etagsim

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

const (
	// InitialETag is the ETag returned on the first GET of the chassis.
	InitialETag = `"v1"`
	jsonODataID = "@odata.id"
)

type chassisResource struct {
	id         string
	name       string
	assetTag   string
	uri        string
	generation int
}

// Simulator is an in-memory Redfish service with ETag / If-Match support.
type Simulator struct {
	mu      sync.Mutex
	chassis map[string]*chassisResource
}

// New returns a simulator with one chassis at /redfish/v1/Chassis/1.
func New() *Simulator {
	return &Simulator{
		chassis: map[string]*chassisResource{
			"1": {
				id:         "1",
				name:       "Computer System Chassis",
				assetTag:   "original",
				uri:        "/redfish/v1/Chassis/1",
				generation: 1,
			},
		},
	}
}

// Handler serves the Redfish endpoints.
func (s *Simulator) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/redfish/v1", s.serviceRoot)
	mux.HandleFunc("/redfish/v1/", s.route)
	return mux
}

func (s *Simulator) route(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	switch path {
	case "/redfish/v1":
		s.serviceRoot(w, r)
	case "/redfish/v1/Chassis":
		s.chassisCollection(w, r)
	default:
		const prefix = "/redfish/v1/Chassis/"
		if strings.HasPrefix(path, prefix) {
			s.chassisMember(w, r, strings.TrimPrefix(path, prefix))
			return
		}
		http.NotFound(w, r)
	}
}

func (s *Simulator) serviceRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		jsonODataID:      "/redfish/v1/",
		"@odata.type":    "#ServiceRoot.v1_5_1.ServiceRoot",
		"Id":             "RootService",
		"Name":           "Root Service",
		"RedfishVersion": "1.15.0",
		"Chassis":        map[string]string{jsonODataID: "/redfish/v1/Chassis"},
	})
}

func (s *Simulator) chassisCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	members := make([]map[string]string, 0, len(s.chassis))
	for _, ch := range s.chassis {
		members = append(members, map[string]string{jsonODataID: ch.uri})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		jsonODataID:           "/redfish/v1/Chassis",
		"@odata.type":         "#ChassisCollection.ChassisCollection",
		"Name":                "Chassis Collection",
		"Members":             members,
		"Members@odata.count": len(members),
	})
}

func (s *Simulator) chassisMember(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.chassis[id]
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		etag := etagValue(ch)
		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		writeJSON(w, http.StatusOK, chassisBody(ch))
	case http.MethodPatch:
		patchChassis(w, r, ch)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func patchChassis(w http.ResponseWriter, r *http.Request, ch *chassisResource) {
	ifMatch := r.Header.Get("If-Match")
	current := etagValue(ch)
	switch {
	case ifMatch == "":
		writeRedfishError(w, http.StatusPreconditionRequired, "Base.1.18.PreconditionRequired", "If-Match header required")
		return
	case ifMatch != current:
		writeRedfishError(w, http.StatusPreconditionFailed, "Base.1.18.PreconditionFailed", "ETag mismatch")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var patch map[string]any
	if err := json.Unmarshal(body, &patch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if tag, ok := patch["AssetTag"].(string); ok {
		ch.assetTag = tag
	}
	ch.generation++
	w.Header().Set("ETag", etagValue(ch))
	w.WriteHeader(http.StatusNoContent)
}

func etagValue(ch *chassisResource) string {
	return fmt.Sprintf(`"v%d"`, ch.generation)
}

func chassisBody(ch *chassisResource) map[string]any {
	return map[string]any{
		jsonODataID:   ch.uri,
		"@odata.type": "#Chassis.v1_0_0.Chassis",
		"Id":          ch.id,
		"Name":        ch.name,
		"ChassisType": "RackMount",
		"AssetTag":    ch.assetTag,
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeRedfishError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
