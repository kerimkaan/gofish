//
// SPDX-License-Identifier: BSD-3-Clause
//

package etagsim

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSimulatorHonorsIfMatch(t *testing.T) {
	ts := httptest.NewServer(New().Handler())
	t.Cleanup(ts.Close)
	ctx := context.Background()
	uri := ts.URL + "/redfish/v1/Chassis/1"

	etag := getETag(t, ctx, ts.Client(), uri)
	if etag != InitialETag {
		t.Fatalf("ETag = %q, want %q", etag, InitialETag)
	}

	stale := patchChassisHTTP(t, ctx, ts.Client(), uri, `"stale"`, "nope")
	if stale != http.StatusPreconditionFailed {
		t.Fatalf("stale If-Match status = %d, want 412", stale)
	}

	matched := patchChassisHTTP(t, ctx, ts.Client(), uri, etag, "ok")
	if matched != http.StatusNoContent {
		t.Fatalf("matching If-Match status = %d, want 204", matched)
	}
}

func getETag(t *testing.T, ctx context.Context, client *http.Client, uri string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.Header.Get("ETag")
}

func patchChassisHTTP(t *testing.T, ctx context.Context, client *http.Client, uri, ifMatch, assetTag string) int {
	t.Helper()
	body := strings.NewReader(`{"AssetTag":"` + assetTag + `"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, uri, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", ifMatch)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PATCH: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode
}
