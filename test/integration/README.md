# ETag / If-Match integration test

Opt-in coverage for gofish reading an `ETag` response header and sending it
back as `If-Match` on `PATCH`. Ordinary `go test ./...` and CI omit this
package (`//go:build integration`).

## Why not a DMTF simulator

Both public DMTF tools were checked against their source and, for
Redfish-Mockup-Server, a live process:

| Service | ETag on GET | If-Match on PATCH |
|---|---|---|
| [Redfish-Mockup-Server](https://github.com/DMTF/Redfish-Mockup-Server) | No. `--test-etag` is documented unimplemented. Default mockup has no `headers.json`. PATCH with a stale or missing `If-Match` still returns 204. | No |
| [Redfish-Interface-Emulator](https://github.com/DMTF/Redfish-Interface-Emulator) | No HTTP `ETag` handling (only a `Settings.ETag` JSON field copied from the mockup). PATCH does not inspect `If-Match`. | No |

Neither actually exercises the behavior, so the test's default target is a
small Go Redfish fake in `internal/etagsim`. It serves a ServiceRoot, a
Chassis collection, and one chassis. GET returns an `ETag` header (and no
`@odata.etag`, so the header path is the one under test). PATCH requires a
matching `If-Match` (412 on mismatch, 428 if omitted) and bumps the tag.
GET with `If-None-Match` returns 304 when the tag is current.

Point `GOFISH_TEST_REDFISH_URL` at a BMC that does implement ETags to run the
same assertions against real firmware. If that service omits ETags, the test
skips with an explanation.

## Run the test (macOS or Linux)

From the repository root, with Go 1.21 or newer:

```bash
make test-integration
```

or:

```bash
go test -tags=integration -count=1 -v ./test/integration
```

The test starts the Go fake on an ephemeral port. No Docker and no extra
environment variables are required.

To drive a service you started yourself:

```bash
# Go fake on :8000
make etag-sim
# in another terminal:
GOFISH_TEST_REDFISH_URL=http://127.0.0.1:8000 make test-integration
```

`etag-sim` accepts `-addr` (default `:8000`).

## DMTF Mockup-Server (optional)

Useful as a general Redfish target. It will not satisfy the ETag assertions
(`GOFISH_TEST_REDFISH_URL` against it skips once GET returns no tag).

Install [Docker Desktop for Mac](https://docs.docker.com/desktop/setup/install/mac-install/), then:

```bash
docker compose -f test/integration/docker-compose.yml up
```

The mockup listens on `http://127.0.0.1:8000`. Without Docker, from a clone of
[Redfish-Mockup-Server](https://github.com/DMTF/Redfish-Mockup-Server):

```bash
pip3 install -r requirements.txt
python3 redfishMockupServer.py -H 127.0.0.1 -p 8000
```

Omitting `-D` serves the bundled `public-rackmount1` mockup.
