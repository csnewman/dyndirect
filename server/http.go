package server

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/json"
	"net/http"
	"time"

	v1 "github.com/csnewman/dyndirect/server/internal/v1"
	oapi "github.com/deepmap/oapi-codegen/pkg/chi-middleware"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func (s *Server) buildHTTPRouter() (*chi.Mux, error) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)

	if s.cfg.APIBehindProxy {
		r.Use(middleware.RealIP)
	}

	r.Use(s.httpLogger)
	r.Use(middleware.NoCache)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(s.httpRecoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	spec, err := v1.GetSwagger()
	if err != nil {
		return nil, err
	}

	r.Use(oapi.OapiRequestValidatorWithOptions(
		spec,
		&oapi.Options{
			Options: openapi3filter.Options{},
			ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
				v := &v1.ErrorResponse{
					Error:   "bad-request",
					Message: message,
				}

				buf := &bytes.Buffer{}
				enc := json.NewEncoder(buf)
				enc.SetEscapeHTML(true)
				if err := enc.Encode(v); err != nil {
					panic(err)
				}

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write(buf.Bytes())
			},
			MultiErrorHandler: nil,
		},
	))

	tokenHash := sha512.Sum512([]byte(s.cfg.TokenKey))

	v1.HandlerWithOptions(
		v1.NewStrictHandlerWithOptions(
			&v1API{
				tokenHash: tokenHash[:],
				store:     s.store,
			},
			[]v1.StrictMiddlewareFunc{
				s.requestMiddleware,
			},
			v1.StrictHTTPServerOptions{
				RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
					s.logger.WithOptions(zap.AddStacktrace(zapcore.NewNopCore())).Errorw(
						"API Request Error",
						"path", r.URL.Path,
						"request_id", middleware.GetReqID(r.Context()),
						"err", err,
					)

					writeResponse(
						w, r, http.StatusBadRequest,
						"bad-request",
						"An error was found in the request",
					)
				},
				ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
					s.logger.WithOptions(zap.AddStacktrace(zapcore.NewNopCore())).Errorw(
						"API Request Error",
						"path", r.URL.Path,
						"request_id", middleware.GetReqID(r.Context()),
						"err", err,
					)

					writeResponse(
						w, r, http.StatusInternalServerError,
						"internal-error",
						"An internal server error has occurred",
					)
				},
			},
		),
		v1.ChiServerOptions{
			BaseRouter: r,
			ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				s.logger.WithOptions(zap.AddStacktrace(zapcore.NewNopCore())).Errorw(
					"API Request Error",
					"path", r.URL.Path,
					"request_id", middleware.GetReqID(r.Context()),
					"err", err,
				)

				writeResponse(
					w, r, http.StatusInternalServerError,
					"internal-error",
					"An internal server error has occurred",
				)
			},
		},
	)

	return r, nil
}

func (s *Server) buildHTTPRedirectServer() *http.Server {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)

	if s.cfg.APIBehindProxy {
		r.Use(middleware.RealIP)
	}

	r.Use(s.httpLogger)
	r.Use(s.httpRecoverer)
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

func (s *Server) buildHTTPServer() (*http.Server, error) {
	hr, err := s.buildHTTPRouter()
	if err != nil {
		return nil, err
	}

	return &http.Server{
		Addr:         s.cfg.APIListenHTTP,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      hr,
	}, nil
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

func (s *Server) httpRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				//nolint:errorlint,goerr113
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}

				s.logger.WithOptions(zap.AddStacktrace(zapcore.NewNopCore())).Errorw(
					"API Request Error",
					"path", r.URL.Path,
					"request_id", middleware.GetReqID(r.Context()),
					"err", rvr,
				)

				writeResponse(
					w, r, http.StatusInternalServerError,
					"internal-error",
					"An internal server error has occurred",
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func writeResponse(w http.ResponseWriter, r *http.Request, status int, code string, msg string) {
	w.WriteHeader(status)
	render.JSON(w, r, &v1.ErrorResponse{
		Error:   code,
		Message: msg,
	})
}

type requestKeyType string

const requestKey requestKeyType = "dd-http-request"

func (s *Server) requestMiddleware(f v1.StrictHandlerFunc, _ string) v1.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, args any) (any, error) {
		return f(context.WithValue(ctx, requestKey, r), w, r, args)
	}
}

func requestFromCtx(ctx context.Context) (*http.Request, bool) {
	u, ok := ctx.Value(requestKey).(*http.Request)

	return u, ok
}
