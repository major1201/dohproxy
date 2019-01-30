package main

import (
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"net"
	"time"
)

// Handler represents how DNS requests be handled
type Handler struct {
	Upstreams map[string]Upstream
	Rules     []Rule
}

// ServeDNS actually handle the DNS requests
func (handler *Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	defer w.Close()

	isMatched := false
	ruleSearchStartTime := time.Now()

	// log fields
	fields := make([]zap.Field, 6)
	fields[0] = zap.String("from", w.RemoteAddr().Network()+"://"+w.RemoteAddr().String())
	fields[1] = zap.String("to", w.LocalAddr().Network()+"://"+w.LocalAddr().String())
	fields[3] = zap.String("question", r.Question[0].String())
	fields[5] = zap.Uint16("id", r.Id)

	for _, rule := range handler.Rules {
		if rule.Upstream() == nil && r.Question[0].Qtype != dns.TypeA {
			continue
		}
		if rule.Matches(r.Question[0].Name) {
			isMatched = true
			if rule.Upstream() == nil {
				fields[2] = zap.String("upstream", "static")
			} else {
				fields[2] = zap.String("upstream", rule.Upstream().Name())
			}
			fields[4] = zap.Duration("searchtime", time.Since(ruleSearchStartTime))
			zap.L().Named("query").Info("routing request", fields[:]...)
			query(rule, w, r)
			break
		}
	}
	if !isMatched {
		fields[2] = zap.String("upstream", "nil")
		fields[4] = zap.Duration("searchtime", time.Since(ruleSearchStartTime))
		zap.L().Named("query").Info("routing request", fields[:]...)
	}
}

func query(r Rule, w dns.ResponseWriter, req *dns.Msg) {
	if len(req.Question) > 1 {
		zap.L().Debug("question number > 1", zap.Int("length", len(req.Question))) // what
	}
	if r.Upstream() != nil {
		r.Upstream().Query(w, req)
		return
	}
	respMsg := &dns.Msg{}
	respMsg.SetReply(req)
	switch req.Question[0].Qtype {
	case dns.TypeA:
		respMsg.Authoritative = true
		domain := req.Question[0].Name
		respMsg.Answer = append(respMsg.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   domain,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.ParseIP(r.StaticResult()),
		})
		w.WriteMsg(respMsg)
	default:
		zap.L().Named("query").Error("static type request qtype must be A at this time")
	}
}
