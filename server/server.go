package server

import (
	"context"
	"crypto/tls"
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
	"net"
	"strings"
)

type Server struct {
	logger *zap.SugaredLogger
	cfg    Config
	acm    *autocert.Manager
}

func New(logger *zap.SugaredLogger, cfg Config) *Server {
	s := &Server{
		logger: logger,
		cfg:    cfg,
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
	group, _ := errgroup.WithContext(context.Background())

	group.Go(func() error {
		return dns.ListenAndServe(":53", "udp", s)
	})

	hs := s.buildHTTPServer()

	if s.cfg.APIListenHTTPS != "" {
		rs := s.buildHTTPRedirectServer()
		group.Go(rs.ListenAndServe)

		hs.Addr = s.cfg.APIListenHTTPS
		hs.TLSConfig = &tls.Config{}

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

func (s *Server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true
	m.Authoritative = true

	if err := s.handleDNS(r, m); err != nil {
		s.logger.Errorw("Failed to process request", "err", err)

		return
	}

	if err := w.WriteMsg(m); err != nil {
		s.logger.Errorw("Failed to process request", "err", err)

		return
	}
}

func (s *Server) handleDNS(r *dns.Msg, m *dns.Msg) error {
	for _, q := range r.Question {
		s.logger.Infow("Question", "Id", r.Id, "Name", q.Name, "Qtype", q.Qtype, "Qclass", q.Qclass)

		if q.Qclass != dns.ClassINET {
			continue
		}

		lcName := strings.ToLower(q.Name)
		var name string

		if lcName == s.cfg.RootDomain {
			name = "@"
		} else if strings.HasSuffix(lcName, "."+s.cfg.RootDomain) {
			name = strings.TrimSuffix(lcName, "."+s.cfg.RootDomain)
		} else {
			continue
		}

		static, hasStatic := s.cfg.StaticRecords[name]

		if hasStatic {
			if q.Qtype == dns.TypeA {
				for _, a := range static.A {
					m.Answer = append(m.Answer, &dns.A{
						Hdr: dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: q.Qclass, Ttl: 0},
						A:   net.ParseIP(a).To4(),
					})
				}
			}

			continue
		}
	}

	return nil
}
