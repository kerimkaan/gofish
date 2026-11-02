//
// SPDX-License-Identifier: BSD-3-Clause
//

// Command etag-sim serves a tiny Redfish mock that returns ETag headers and
// honors If-Match on PATCH. Use it to exercise gofish's ETag handling locally.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/stmcginnis/gofish/internal/etagsim"
)

func main() {
	addr := flag.String("addr", ":8000", "listen address")
	flag.Parse()

	server := &http.Server{
		Addr:              *addr,
		Handler:           etagsim.New().Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("etag-aware Redfish simulator listening on %s", *addr)
	log.Fatal(server.ListenAndServe())
}
