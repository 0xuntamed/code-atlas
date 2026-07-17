package api

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func (s *Server) security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			writeError(
				w,
				http.StatusForbidden,
				"invalid_host",
				"CodeAtlas accepts loopback requests only",
			)
			return
		}

		if origin := r.Header.Get("Origin"); origin != "" && !isLoopbackOrigin(origin) {
			writeError(
				w,
				http.StatusForbidden,
				"invalid_origin",
				"Cross-origin request denied",
			)
			return
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set(
			"Content-Security-Policy",
			"default-src 'self'; worker-src 'self' blob:; style-src 'self' 'unsafe-inline'",
		)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func isLoopbackHost(host string) bool {
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		host = hostname
	}
	host = strings.Trim(host, "[]")
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func isLoopbackOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
