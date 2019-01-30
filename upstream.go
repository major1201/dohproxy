package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// Upstream describes an upstream interface
type Upstream interface {
	Type() string
	Name() string
	Query(w dns.ResponseWriter, req *dns.Msg)
}

// UpstreamImpl describes an abstract implement of the Upstream interface
type UpstreamImpl struct {
	name    string
	address string
}

// Name returns the upstream name
func (u *UpstreamImpl) Name() string {
	return u.name
}

// Address returns the upstream address
func (u *UpstreamImpl) Address() string {
	return u.address
}

// UpstreamDNS is a DNS upstream
type UpstreamDNS struct {
	UpstreamImpl
}

// UpstreamDoh is an abstract DNS-over-HTTPS upstream
type UpstreamDoh struct {
	UpstreamImpl
	proxy *url.URL
}

// UpstreamDohGet is the DNS-over-HTTPS upstream implement using HTTP GET method
type UpstreamDohGet struct {
	UpstreamDoh
}

// UpstreamDohPost is the DNS-over-HTTPS upstream implement using HTTP POST method
type UpstreamDohPost struct {
	UpstreamDoh
}

// UpstreamBlackHole does nothing to all DNS requests
type UpstreamBlackHole struct{}

// UpstreamReject returns error the all DNS requests
type UpstreamReject struct{}

// Type returns the type of the dns upstream
func (upstream *UpstreamDNS) Type() string {
	return "dns"
}

// Type returns the type of the DNS-over-HTTPS upstream using HTTP GET method
func (upstream *UpstreamDohGet) Type() string {
	return "doh-get"
}

// Type returns the type of the DNS-over-HTTPS upstream using HTTP POST method
func (upstream *UpstreamDohPost) Type() string {
	return "doh-post"
}

// Type returns the type of the black hole upstream
func (upstream *UpstreamBlackHole) Type() string {
	return "blackhole"
}

// Type returns the type of the reject upstream
func (upstream *UpstreamReject) Type() string {
	return "reject"
}

// Name returns the black hole upstream name
func (upstream *UpstreamBlackHole) Name() string {
	return "blackhole"
}

// Name returns the reject upstream name
func (upstream *UpstreamReject) Name() string {
	return "reject"
}

func (upstream *UpstreamDoh) dohQuery(w dns.ResponseWriter, req *dns.Msg, method string) {
	logger := zap.L().Named("answer").With(zap.Uint16("id", req.Id))

	u, err := url.Parse(upstream.address)
	if err != nil {
		logger.Fatal("doh url parse error", zap.Error(err))
	}

	// Update TLS and HTTP client configuration
	tlsConfig := &tls.Config{ServerName: u.Hostname()}
	transport := &http.Transport{
		TLSClientConfig:    tlsConfig,
		DisableCompression: true,
		MaxIdleConns:       1,
	}
	if upstream.proxy != nil {
		transport.Proxy = http.ProxyURL(upstream.proxy)
	}
	http2.ConfigureTransport(transport)

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	// start
	msg, err := req.Pack()
	if err != nil {
		logger.Error("doh pack message error", zap.Error(err))
		return
	}

	var httpReq *http.Request
	switch method {
	case "GET":
		base64str := base64.RawURLEncoding.EncodeToString(msg)
		httpReq, err = http.NewRequest("GET", u.String()+"?dns="+base64str, bytes.NewBuffer(msg))
	case "POST":
		httpReq, err = http.NewRequest("POST", u.String(), bytes.NewBuffer(msg))
	default:
		zap.L().Fatal("illegal http method", zap.String("method", method))
	}

	if err != nil {
		logger.Error("doh req err", zap.Error(err))
		return
	}
	httpReq.Header.Add("Content-Type", "application/dns-message")
	httpReq.Host = u.Hostname()

	httpResp, err := client.Do(httpReq)
	if err != nil {
		logger.Warn("doh resp err", zap.Error(err))
		return
	}

	defer httpResp.Body.Close()

	switch httpResp.StatusCode {
	case http.StatusOK: // 200
		buf, err := ioutil.ReadAll(httpResp.Body)
		if err != nil {
			logger.Error("doh read http body error", zap.Error(err))
			return
		}
		w.Write(buf)
	case http.StatusBadRequest: // 400
		logger.Info("DNS query not specified or too small.")
	case http.StatusRequestEntityTooLarge: // 413
		logger.Info("DNS query is larger than maximum allowed DNS message size.")
	case http.StatusUnsupportedMediaType: // 415
		logger.Info("Unsupported content type.")
	case http.StatusGatewayTimeout: // 504
		logger.Info("Resolver timeout while waiting for the query response.")
	default:
		logger.Info("Unknown http status code", zap.Int("code", httpResp.StatusCode))
	}
}

// Query does the exact query action of an DNS upstream
func (upstream *UpstreamDNS) Query(w dns.ResponseWriter, req *dns.Msg) {
	c := new(dns.Client)
	r, _, err := c.Exchange(req, upstream.address)
	if err != nil {
		zap.L().Named("answer").Warn("exchange dns server failed",
			zap.Uint16("id", req.Id),
		)
		return
	}
	w.WriteMsg(r)
}

// Query does the exact query action of an DNS-over-HTTPS upstream using HTTP GET method
func (upstream *UpstreamDohGet) Query(w dns.ResponseWriter, req *dns.Msg) {
	upstream.dohQuery(w, req, "GET")
}

// Query does the exact query action of an DNS-over-HTTPS upstream using HTTP POST method
func (upstream *UpstreamDohPost) Query(w dns.ResponseWriter, req *dns.Msg) {
	upstream.dohQuery(w, req, "POST")
}

// Query does the exact query action of an black hole upstream
func (upstream *UpstreamBlackHole) Query(w dns.ResponseWriter, req *dns.Msg) {
	// just do nothing
}

// Query does the exact query action of an reject upstream
func (upstream *UpstreamReject) Query(w dns.ResponseWriter, req *dns.Msg) {
	msg := &dns.Msg{}
	msg.SetReply(req)
	w.WriteMsg(msg)
}
