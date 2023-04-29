package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"time"
)

func (s *Server) buildHTTPRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)

	if s.cfg.APIBehindProxy {
		r.Use(middleware.RealIP)
	}

	r.Use(s.httpLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("TODO"))
	})

	return r
}

func (s *Server) buildHTTPRedirectServer() *http.Server {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)

	if s.cfg.APIBehindProxy {
		r.Use(middleware.RealIP)
	}

	r.Use(s.httpLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	if s.cfg.ACMEEnabled {
		r.Use(s.acm.HTTPHandler)
	}

	r.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL
		u.Scheme = "https"
		u.Host = s.cfg.APIHost
		http.Redirect(w, r, u.String(), http.StatusPermanentRedirect)
	}))

	return &http.Server{
		Addr:         s.cfg.APIListenHTTP,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
		Handler:      r,
	}
}

func (s *Server) buildHTTPServer() *http.Server {
	hr := s.buildHTTPRouter()

	return &http.Server{
		Addr:         s.cfg.APIListenHTTP,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      hr,
	}
}

func (s *Server) httpLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		t1 := time.Now()
		defer func() {
			ua := ww.Header().Get("User-Agent")
			if ua == "" {
				ua = r.Header.Get("User-Agent")
			}

			s.logger.Debugw(
				"Served API Request",
				"path", r.URL.Path,
				"request_id", middleware.GetReqID(r.Context()),
				"took", time.Since(t1),
				"status", ww.Status(),
				"size", ww.BytesWritten(),
				"ua", ua,
			)
		}()
		next.ServeHTTP(ww, r)
	})
}
