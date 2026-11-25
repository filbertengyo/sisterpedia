package handlers

import (
	"context"
	"net/http"
	"runtime/debug"
	"slices"
	"time"

	"github.com/filbertengyo/sisterpedia/internal/server/service"
	"github.com/go-chi/chi/v5"
)

type logInfoKey struct{}

type logInfo struct{ user string }

func setUser(r *http.Request, id string) {
	if li, ok := r.Context().Value(logInfoKey{}).(*logInfo); ok {
		li.user = id
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// accessLog logs method, route, status, duration, user_id and sizes only.
// Bodies (ciphertexts, keys) are never logged, and there is no plaintext.
func (a *api) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		li := &logInfo{}
		sw := &statusWriter{ResponseWriter: w}
		r = r.WithContext(context.WithValue(r.Context(), logInfoKey{}, li))
		next.ServeHTTP(sw, r)
		route := r.URL.Path
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			route = rc.RoutePattern()
			if id := rc.URLParam("user_id"); id != "" && li.user == "" {
				li.user = id
			}
		}
		user := li.user
		if user == "" {
			user = "-"
		}
		a.log.Printf("%s %s status=%d dur=%s user=%s req_bytes=%d res_bytes=%d",
			r.Method, route, sw.status, time.Since(start).Round(time.Microsecond), user, r.ContentLength, sw.bytes)
	})
}

func (a *api) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler { //nolint:errorlint // sentinel comparison required by net/http
					panic(rec)
				}
				a.log.Printf("panic: %v\n%s", rec, debug.Stack())
				writeError(w, http.StatusInternalServerError, service.CodeInternal, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// cors allows only the configured GUI origin(s), methods GET/POST/OPTIONS
// and the Content-Type header. Preflight requests are answered here.
func (a *api) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && slices.Contains(a.opts.AllowedOrigins, origin)
		if origin != "" {
			w.Header().Add("Vary", "Origin")
		}
		if allowed {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Expose-Headers", "Server-Timing")
			h.Set("Timing-Allow-Origin", origin)
		}
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			if allowed {
				h := w.Header()
				h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type")
				h.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Cache-Control", "no-store")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
