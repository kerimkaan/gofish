//
// SPDX-License-Identifier: BSD-3-Clause
//

//go:build integration

package integration_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/internal/etagsim"
	"github.com/stmcginnis/gofish/schemas"
)

// TestETagIfMatchEndToEnd drives gofish against a Redfish HTTP service and
// checks that a GET captures the ETag header and that the following PATCH
// sends If-Match. Point GOFISH_TEST_REDFISH_URL at a live service to use that
// instead of the local etag-aware simulator.
func TestETagIfMatchEndToEnd(t *testing.T) {
	endpoint, live := integrationEndpoint(t)

	var dump bytes.Buffer
	client, err := gofish.Connect(gofish.ClientConfig{
		Endpoint:   endpoint,
		DumpWriter: &dump,
		Insecure:   true,
	})
	if err != nil {
		t.Fatalf("Connect(%s): %v", endpoint, err)
	}
	defer client.Logout()

	ch := firstChassis(t, client, live)
	etag := requireETag(t, ch, live)

	updatedTag := fmt.Sprintf("gofish-etag-%d", time.Now().UnixNano())
	patchWithIfMatch(t, ch, updatedTag, &dump)
	assertStaleIfMatch(t, ch, updatedTag+"-stale", live)
	assertPatchedChassis(t, client, ch.ODataID, updatedTag, etag, live)
}

func integrationEndpoint(t *testing.T) (string, bool) {
	t.Helper()
	if url := os.Getenv("GOFISH_TEST_REDFISH_URL"); url != "" {
		return url, true
	}
	ts := httptest.NewServer(etagsim.New().Handler())
	t.Cleanup(ts.Close)
	return ts.URL, false
}

func firstChassis(t *testing.T, client *gofish.APIClient, live bool) *schemas.Chassis {
	t.Helper()
	list, err := client.Service.Chassis()
	if err != nil {
		t.Fatalf("Chassis(): %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected at least one Chassis")
	}
	ch := list[0]
	if ch.GetETag() == "" && live {
		t.Skip("live service did not return an ETag header or @odata.etag; " +
			"DMTF Redfish-Mockup-Server leaves --test-etag unimplemented and " +
			"does not send ETag. Omit GOFISH_TEST_REDFISH_URL to use the local simulator.")
	}
	return ch
}

func requireETag(t *testing.T, ch *schemas.Chassis, live bool) string {
	t.Helper()
	etag := ch.GetETag()
	if etag == "" {
		t.Fatal("GetETag() is empty after GET; want the ETag response header")
	}
	if !live && etag != etagsim.InitialETag {
		t.Fatalf("GetETag() = %q, want initial simulator etag %q", etag, etagsim.InitialETag)
	}
	return etag
}

func patchWithIfMatch(t *testing.T, ch *schemas.Chassis, assetTag string, dump *bytes.Buffer) {
	t.Helper()
	ch.AssetTag = assetTag
	if err := ch.Update(); err != nil {
		t.Fatalf("Update() with captured ETag: %v", err)
	}
	if !strings.Contains(dump.String(), "If-Match:") {
		t.Fatal("PATCH was sent without an If-Match header")
	}
}

func assertStaleIfMatch(t *testing.T, ch *schemas.Chassis, assetTag string, live bool) {
	t.Helper()
	ch.AssetTag = assetTag
	err := ch.Update()
	switch {
	case schemas.Is412PreconditionFailed(err):
		return
	case err == nil && live:
		t.Log("live service accepted a PATCH with a stale If-Match; " +
			"DMTF Mockup-Server and Interface-Emulator do not enforce If-Match")
	case err == nil:
		t.Fatal("stale If-Match: Update() succeeded, want 412 Precondition Failed")
	default:
		t.Fatalf("stale If-Match: Update() error %v, want 412 Precondition Failed", err)
	}
}

func assertPatchedChassis(t *testing.T, client *gofish.APIClient, uri, wantTag, oldETag string, live bool) {
	t.Helper()
	fresh, err := schemas.GetChassis(client, uri)
	if err != nil {
		t.Fatalf("re-GET chassis: %v", err)
	}
	if fresh.AssetTag != wantTag {
		t.Fatalf("AssetTag after PATCH = %q, want %q", fresh.AssetTag, wantTag)
	}
	if !live && fresh.GetETag() == oldETag {
		t.Fatalf("ETag still %q after a successful PATCH, want it to change", oldETag)
	}

	refreshed, err := schemas.Refresh(fresh)
	switch {
	case errors.Is(err, schemas.ErrNotModified):
		if refreshed.AssetTag != wantTag {
			t.Fatalf("Refresh 304 returned AssetTag %q, want %q", refreshed.AssetTag, wantTag)
		}
	case live && err == nil:
		t.Log("live service does not honor If-None-Match (no 304)")
	case err != nil:
		t.Fatalf("Refresh(): %v", err)
	}
}
