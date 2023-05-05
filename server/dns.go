package server

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func (s *Server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	defer func() {
		if rvr := recover(); rvr != nil {
			s.logger.WithOptions(zap.AddStacktrace(zapcore.NewNopCore())).Errorw(
				"DNS Request Error",
				"request_id", r.Id,
				"err", rvr,
			)
		}
	}()

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true
	m.Authoritative = true

	if err := s.handleDNS(r, m); err != nil {
		s.logger.WithOptions(zap.AddStacktrace(zapcore.NewNopCore())).Errorw(
			"DNS Request Error",
			"request_id", r.Id,
			"err", err,
		)

		return
	}

	if err := w.WriteMsg(m); err != nil {
		s.logger.WithOptions(zap.AddStacktrace(zapcore.NewNopCore())).Errorw(
			"DNS Request Error",
			"request_id", r.Id,
			"err", err,
		)

		return
	}
}

func (s *Server) handleDNS(r *dns.Msg, m *dns.Msg) error {
	for _, q := range r.Question {
		s.logger.Infow("DNS Question", "Id", r.Id, "Name", q.Name, "Qtype", q.Qtype, "Qclass", q.Qclass)

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

		parts := strings.Split(name, ".")
		if len(parts) != 2 {
			continue
		}

		id, err := uuid.Parse(parts[1])
		if err != nil {
			continue
		}

		req := parts[0]

		if req == "_acme-challenge" && q.Qtype == dns.TypeTXT {
			s.logger.Infow("DNS ACME Request", "name", q.Name, "id", id)

			ctx, can := context.WithTimeout(context.Background(), time.Second*5)
			defer can()

			values, err := s.store.GetACMEChallengeTokens(ctx, id)
			if err != nil {
				return err
			}

			for _, token := range values {
				m.Answer = append(m.Answer, &dns.TXT{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: q.Qclass, Ttl: 0},
					Txt: []string{token},
				})
			}

			continue
		}

		lastInd := strings.LastIndex(req, "-")
		if lastInd == -1 {
			continue
		}

		reqType := req[lastInd+1:]
		reqValue := req[:lastInd]

		if reqType == "v4" && q.Qtype == dns.TypeA {
			v4 := net.ParseIP(strings.ReplaceAll(reqValue, "-", "."))
			if v4 == nil {
				continue
			}

			v4 = v4.To4()
			if v4 == nil {
				continue
			}

			m.Answer = append(m.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: q.Qclass, Ttl: 0},
				A:   v4,
			})

			s.logger.Infow("DNS V4 Request", "name", q.Name, "id", id, "ip", v4)

			continue
		} else if reqType == "v6" && q.Qtype == dns.TypeAAAA {
			v6 := net.ParseIP(strings.ReplaceAll(reqValue, "-", ":"))
			if v6 == nil {
				continue
			}

			v6 = v6.To16()
			if v6 == nil {
				continue
			}

			m.Answer = append(m.Answer, &dns.AAAA{
				Hdr:  dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: q.Qclass, Ttl: 0},
				AAAA: v6,
			})

			s.logger.Infow("DNS V6 Request", "name", q.Name, "id", id, "ip", v6)

			continue
		}
	}

	return nil
}
