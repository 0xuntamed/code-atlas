package api

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func (s *Server) security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.hostAllowed(r.Host) {
			writeError(
				w,
				http.StatusForbidden,
				"invalid_host",
				"CodeAtlas accepts loopback requests only",
			)
			return
		}

		if origin := r.Header.Get("Origin"); origin != "" && !s.originAllowed(origin) {
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
			"default-src 'self'; worker-src 'self' blob:; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
				"font-src 'self' https://fonts.gstatic.com",
		)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// hostAllowed accepts loopback always, plus any host in CODEATLAS_ALLOWED_HOSTS
// (or everything when that list contains "*"), so a deployed instance can serve
// its public domain while local runs stay loopback-only.
func (s *Server) hostAllowed(host string) bool {
	if isLoopbackHost(host) {
		return true
	}
	if s.allowAllHosts {
		return true
	}
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		host = hostname
	}
	host = strings.ToLower(strings.Trim(host, "[]"))
	return s.allowedHosts[host]
}

// originAllowed applies the same rule to the Origin header on cross-origin
// requests (the browser omits it for same-origin GETs).
func (s *Server) originAllowed(origin string) bool {
	if isLoopbackOrigin(origin) {
		return true
	}
	if s.allowAllHosts {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return s.allowedHosts[strings.ToLower(parsed.Hostname())]
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
