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
	isMatched := false
	ruleSearchStartTime := time.Now()
	for _, rule := range handler.Rules {
		if rule.Matches(r.Question[0].Name) {
			isMatched = true
			query(rule, w, r, time.Since(ruleSearchStartTime))
			break
		}
	}
	if !isMatched {
		zap.L().Named("query").Info("no rule matches domain", zap.String("domain", r.Question[0].Name))
	}
}

func query(r Rule, w dns.ResponseWriter, req *dns.Msg, ruleSearchTime time.Duration) {
	if len(req.Question) > 1 {
		zap.L().Debug("question number > 1", zap.Int("length", len(req.Question))) // what
	}
	if r.Upstream() != nil {
		zap.L().Named("query").Info("routing request",
			zap.String("from", w.RemoteAddr().Network()+"://"+w.RemoteAddr().String()),
			zap.String("to", w.LocalAddr().Network()+"://"+w.LocalAddr().String()),
			zap.String("upstream", r.Upstream().Name()),
			zap.String("question", req.Question[0].String()),
			zap.Duration("searchtime", ruleSearchTime),
			zap.Uint16("id", req.Id),
		)
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
		zap.L().Named("query").Info("routing request",
			zap.String("from", w.RemoteAddr().String()),
			zap.String("upstream", "static"),
			zap.String("question", req.Question[0].String()),
			zap.String("answer", r.StaticResult()),
			zap.Duration("searchtime", ruleSearchTime),
			zap.Uint16("id", req.Id),
		)
		w.WriteMsg(respMsg)
	default:
		zap.L().Named("query").Error("static type request qtype must be A at this time")
	}
}
