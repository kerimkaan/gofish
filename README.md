 # Gofish - Redfish and Swordfish client library

[![Go Doc](https://godoc.org/github.com/stmcginnis/gofish?status.svg)](http://godoc.org/github.com/stmcginnis/gofish)
[![Go Report Card](https://goreportcard.com/badge/github.com/stmcginnis/gofish?branch=main)](https://goreportcard.com/report/github.com/stmcginnis/gofish)
[![Releases](https://img.shields.io/github/release/stmcginnis/gofish/all.svg?style=flat-square)](https://github.com/stmcginnis/gofish/releases)
[![LICENSE](https://img.shields.io/github/license/stmcginnis/gofish.svg?style=flat-square)](https://github.com/stmcginnis/gofish/blob/main/LICENSE)

![Gofish Logo](./images/gofish200x117.png)

## Introduction

Gofish is a Golang library for interacting with [DMTF
Redfish](https://www.dmtf.org/standards/redfish) and [SNIA
Swordfish](https://www.snia.org/forums/smi/swordfish) enabled devices.

## Usage ##

Basic usage would be:

```go

package main

import (
    "fmt"

    "github.com/stmcginnis/gofish"
)

func main() {
    c, err := gofish.ConnectDefault("http://localhost:5000")
    if err != nil {
        panic(err)
    }

    service := c.Service
    chassis, err := service.Chassis()
    if err != nil {
        panic(err)
    }

    for _, chass := range chassis {
        fmt.Printf("Chassis: %#v\n\n", chass)
    }
}
```

## Testing ETag / If-Match locally

`go test ./...` stays unit-test only. To exercise GET `ETag` capture and PATCH
`If-Match` against an HTTP service:

```bash
make test-integration
```

That command is build-tagged (`integration`) and talks to a small Go Redfish
fake that actually returns ETags and rejects a stale `If-Match`. The DMTF
Redfish-Mockup-Server and Redfish-Interface-Emulator do not (Mockup-Server's
`--test-etag` flag is unimplemented). See [test/integration/README.md](test/integration/README.md)
for Docker Compose, `make etag-sim`, and pointing `GOFISH_TEST_REDFISH_URL` at
a BMC.
