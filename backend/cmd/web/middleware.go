package main

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

const rateLimiterCleanupInterval = 10 * time.Minute

// cleanupRateLimiters removes idle rate limiters from the sync.Map every rateLimiterCleanupInterval
// to prevent unbounded memory growth as unique IPs accumulate.
func (app *application) cleanupRateLimiters() {
	ticker := time.NewTicker(rateLimiterCleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		app.rateLimiters.Range(func(key, val any) bool {
			if val.(*rate.Limiter).Tokens() >= float64(val.(*rate.Limiter).Burst()) {
				app.rateLimiters.Delete(key)
			}
			return true
		})
	}
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// https://owasp.org/www-project-secure-headers/index.html#configuration-proposal

		// Add
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; form-action 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; upgrade-insecure-requests; block-all-mixed-content")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("Referrer-Policy", "no-referrer")

		// Should be called on logout
		// w.Header().Set("Clear-Site-Data", "cache")
		// w.Header().Set("Clear-Site-Data", "cookies")
		// w.Header().Set("Clear-Site-Data", "storage")

		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "accelerometer=(), autoplay=(), camera=(), cross-origin-isolated=(), display-capture=(), encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), keyboard-map=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), screen-wake-lock=(), sync-xhr=(self), usb=(), web-share=(), xr-spatial-tracking=(), clipboard-read=(), clipboard-write=(), gamepad=(), hid=(), idle-detection=(), interest-cohort=(), serial=(), unload=()")
		w.Header().Set("Cache-Control", "no-store, max-age=0") // Not for SSE

		// Remove, most of these should not be needed, should take a look at which ones.
		w.Header().Del("Server")
		w.Header().Del("Liferay-Portal")
		w.Header().Del("X-Turbo-Charged-By")
		w.Header().Del("X-Powered-By")
		w.Header().Del("X-Server-Powered-By")
		w.Header().Del("X-Powered-CMS")
		w.Header().Del("SourceMap")
		w.Header().Del("X-SourceMap")
		w.Header().Del("X-AspNetMvc-Version")
		w.Header().Del("X-AspNet-Version")
		w.Header().Del("X-SourceFiles")
		w.Header().Del("X-Redirect-By")
		w.Header().Del("X-Generator")
		w.Header().Del("X-Generated-By")
		w.Header().Del("X-CMS")
		w.Header().Del("X-Powered-By-Plesk")
		w.Header().Del("X-Php-Version")
		w.Header().Del("Powered-By")
		w.Header().Del("X-Content-Encoded-By")
		w.Header().Del("Product")
		w.Header().Del("X-CF-Powered-By")
		w.Header().Del("X-Framework")
		w.Header().Del("Host-Header")
		w.Header().Del("Pega-Host")
		w.Header().Del("X-Atmosphere-first-request")
		w.Header().Del("X-Atmosphere-tracking-id")
		w.Header().Del("X-Atmosphere-error")
		w.Header().Del("X-Mod-Pagespeed")
		w.Header().Del("X-Page-Speed")
		w.Header().Del("X-Varnish-Backend")
		w.Header().Del("X-Varnish-Server")
		w.Header().Del("X-Envoy-Upstream-Service-Time")
		w.Header().Del("X-Envoy-Attempt-Count")
		w.Header().Del("X-Envoy-External-Address")
		w.Header().Del("X-Envoy-Internal")
		w.Header().Del("X-Envoy-Original-Dst-Host")
		w.Header().Del("X-B3-ParentSpanId")
		w.Header().Del("X-B3-Sampled")
		w.Header().Del("X-B3-SpanId")
		w.Header().Del("X-B3-TraceId")
		w.Header().Del("K-Proxy-Request")
		w.Header().Del("X-Old-Content-Length")
		w.Header().Del("$wsep")
		w.Header().Del("X-Nextjs-Matched-Path")
		w.Header().Del("X-Nextjs-Page")
		w.Header().Del("X-Nextjs-Cache")
		w.Header().Del("X-Nextjs-Redirect")
		w.Header().Del("X-OneAgent-JS-Injection")
		w.Header().Del("X-ruxit-JS-Agent")
		w.Header().Del("X-dtHealthCheck")
		w.Header().Del("X-dtAgentId")
		w.Header().Del("X-dtInjectedServlet")
		w.Header().Del("X-Litespeed-Cache-Control")
		w.Header().Del("X-LiteSpeed-Purge")
		w.Header().Del("X-LiteSpeed-Tag")
		w.Header().Del("X-LiteSpeed-Vary")
		w.Header().Del("X-LiteSpeed-Cache")
		w.Header().Del("X-Umbraco-Version")
		w.Header().Del("OracleCommerceCloud-Version")
		w.Header().Del("X-BEServer")
		w.Header().Del("X-DiagInfo")
		w.Header().Del("X-FEServer")
		w.Header().Del("X-CalculatedBETarget")
		w.Header().Del("X-OWA-Version")
		w.Header().Del("X-Cocoon-Version")
		w.Header().Del("X-Kubernetes-PF-FlowSchema-UI")
		w.Header().Del("X-Kubernetes-PF-PriorityLevel-UID")
		w.Header().Del("X-Jitsi-Release")
		w.Header().Del("X-Joomla-Version")
		w.Header().Del("X-Backside-Transport")

		next.ServeHTTP(w, r)

	})
}

func (app *application) corsHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", app.allowedOrigin)
		w.Header().Set("Access-Control-Max-Age", "10")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.logger.Info("request", "remoteAddr", r.RemoteAddr, "proto", r.Proto, "method", r.Method, "uri", r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, fmt.Errorf("%v", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func requireLocalhost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host != "127.0.0.1" && host != "::1" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		val, _ := app.rateLimiters.LoadOrStore("general:"+ip, rate.NewLimiter(rate.Every(time.Second), 10)) // 10 per second
		if !val.(*rate.Limiter).Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authRateLimit applies a strict per-IP limit suitable for login/register (5 per minute).
func (app *application) authRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		val, _ := app.rateLimiters.LoadOrStore("auth:"+ip, rate.NewLimiter(rate.Every(12*time.Second), 5)) // 5 per minute
		if !val.(*rate.Limiter).Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) requireSameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodOptions {
			origin := r.Header.Get("Origin")
			if origin != "" && origin != app.allowedOrigin {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			// Skip CSRF check for validateSession — it's what issues the token
			if r.URL.Path != "/validateSession" {
				token := r.Header.Get("X-CSRF-Token")
				sessionToken := app.sessionManager.GetString(r.Context(), "csrfToken")
				if sessionToken == "" || token != sessionToken {
					http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
