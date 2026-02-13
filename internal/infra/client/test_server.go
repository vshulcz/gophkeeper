//go:build integration
// +build integration

package client

import (
	"net/http"
	"net/http/httptest"
)

func newTestServer(handler http.Handler) *httptest.Server {
	return httptest.NewServer(handler)
}
