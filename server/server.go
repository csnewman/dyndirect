package server

import (
	"context"
	"crypto/tls"

	"github.com/miekg/dns"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
)

const Version = "1.0.0"

type Server struct {
	logger *zap.SugaredLogger
	cfg    Config
	acm    *autocert.Manager
	store  Store
}

func New(logger *zap.SugaredLogger, cfg Config, store Store) *Server {
	s := &Server{
		logger: logger,
		cfg:    cfg,
		store:  store,
	}

	if cfg.ACMEEnabled {
		s.acm = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(s.cfg.APIHost),
			Cache:      autocert.DirCache("cache"),
			Email:      s.cfg.ACMEContact,
		}
	}

	return s
}

func (s *Server) Start() error {
	hs, err := s.buildHTTPServer()
	if err != nil {
		return err
	}

	group, _ := errgroup.WithContext(context.Background())

	group.Go(func() error {
		return dns.ListenAndServe(":53", "udp", s)
	})

	if s.cfg.APIListenHTTPS != "" {
		rs := s.buildHTTPRedirectServer()
		group.Go(rs.ListenAndServe)

		hs.Addr = s.cfg.APIListenHTTPS
		hs.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		if s.cfg.ACMEEnabled {
			hs.TLSConfig.GetCertificate = s.acm.GetCertificate
		}

		group.Go(func() error {
			return hs.ListenAndServeTLS(s.cfg.CertFile, s.cfg.KeyFile)
		})
	} else {
		group.Go(hs.ListenAndServe)
	}

	return group.Wait()
}
