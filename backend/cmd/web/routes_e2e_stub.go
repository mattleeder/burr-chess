//go:build !e2e

package main

import "net/http"

// registerE2ERoutes is a no-op in production builds.
func registerE2ERoutes(_ *http.ServeMux, _ *application) {}
