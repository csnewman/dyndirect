package server

import (
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"net"
	"strings"
)

type Server struct {
	logger *zap.SugaredLogger
	cfg    Config
}

func New(logger *zap.SugaredLogger, cfg Config) *Server {
	return &Server{
		logger: logger,
		cfg:    cfg,
	}
}

func (s *Server) Start() error {
	return dns.ListenAndServe(":53", "udp", s)
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

		var name string

		if q.Name == s.cfg.RootDomain {
			name = "@"
		} else if strings.HasSuffix(q.Name, "."+s.cfg.RootDomain) {
			name = strings.TrimSuffix(q.Name, "."+s.cfg.RootDomain)
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
