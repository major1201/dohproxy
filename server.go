package main

import (
	"github.com/go-yaml/yaml"
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"net/url"
	"os"
	"time"
)

// Server describes the server interface
type Server interface {
	Serve() error
	Type() string
	Address() string
	Handler() *Handler
	SetHandler(handler *Handler)
}

// ServerImpl implements the Server interface
type ServerImpl struct {
	address string
	handler *Handler
}

// Address returns the address of a server
func (s *ServerImpl) Address() string {
	return s.address
}

// Handler returns the handler of a server
func (s *ServerImpl) Handler() *Handler {
	return s.handler
}

// SetHandler set the server handler attribute
func (s *ServerImpl) SetHandler(handler *Handler) {
	s.handler = handler
}

// UDPServer is a implement of server using UDP protocol
type UDPServer struct {
	ServerImpl
}

// TCPServer is a implement of server using TCP protocol
type TCPServer struct {
	ServerImpl
}

// Serve starts the UDP DNS server
func (s *UDPServer) Serve() error {
	srv := &dns.Server{
		Addr:    s.address,
		Net:     "udp",
		Handler: s.handler,
	}
	zap.L().Named("server").Info("listening and serving",
		zap.String("proto", "udp"),
		zap.String("address", s.address),
	)
	return srv.ListenAndServe()
}

// Type returns a UDP server type
func (s *UDPServer) Type() string {
	return "udp"
}

// Serve starts the TCP DNS server
func (s *TCPServer) Serve() error {
	srv := &dns.Server{
		Addr:    s.address,
		Net:     "tcp",
		Handler: s.handler,
	}
	zap.L().Named("server").Info("listening and serving",
		zap.String("proto", "tcp"),
		zap.String("address", s.address),
	)
	return srv.ListenAndServe()
}

// Type returns a TCP server type
func (s *TCPServer) Type() string {
	return "tcp"
}

// Config describes the config file
type Config struct {
	Listen    []map[string]string
	Upstreams map[string]map[string]string
	Rules     []string
}

func checkMapAttrs(m map[string]string, parentKey string, keys ...string) {
	for _, key := range keys {
		if _, ok := m[key]; !ok {
			zap.L().Named("config").Fatal("lost key", zap.String("field", parentKey), zap.String("lost key", key))
		}
	}

}

// LoadServersFromConfig loads the config file in YAML format into Server slice objects
func LoadServersFromConfig(configPath string) []Server {
	zap.L().Named("config").Info("reading config file", zap.String("filename", configPath))
	startTime := time.Now()

	yamlFile, err := os.Open(configPath)
	defer yamlFile.Close()
	if err != nil {
		zap.L().Named("config").Fatal("can't open config file", zap.String("filename", configPath))
	}

	configMap := &Config{}
	yaml.NewDecoder(yamlFile).Decode(configMap)

	// upstreams
	handler := &Handler{
		Upstreams: map[string]Upstream{
			"blackhole": &UpstreamBlackHole{},
			"reject":    &UpstreamReject{},
		},
		Rules: []Rule{},
	}
	for name, upstreamConfig := range configMap.Upstreams {
		checkMapAttrs(upstreamConfig, "upstream", "type", "address")

		switch upstreamConfig["type"] {
		case "dns":
			upstream := &UpstreamDNS{
				UpstreamImpl{
					name:    name,
					address: upstreamConfig["address"],
				},
			}
			handler.Upstreams[name] = upstream
		case "doh", "doh-get":
			upstream := &UpstreamDohGet{
				UpstreamDoh{
					UpstreamImpl: UpstreamImpl{
						name:    name,
						address: upstreamConfig["address"],
					},
				},
			}
			if proxyStr, ok := upstreamConfig["proxy"]; ok {
				proxyURL, err := url.Parse(proxyStr)
				if err != nil {
					zap.L().Named("config").Fatal("upstream proxy url parse error", zap.String("upstream name", name), zap.String("proxy", proxyStr))
				}
				upstream.proxy = proxyURL
			}
			handler.Upstreams[name] = upstream
		case "doh-post":
			upstream := &UpstreamDohPost{
				UpstreamDoh{
					UpstreamImpl: UpstreamImpl{
						name:    name,
						address: upstreamConfig["address"],
					},
				},
			}
			if proxyStr, ok := upstreamConfig["proxy"]; ok {
				proxyURL, err := url.Parse(proxyStr)
				if err != nil {
					zap.L().Named("config").Fatal("upstream proxy url parse error", zap.String("upstream name", name), zap.String("proxy", proxyStr))
				}
				upstream.proxy = proxyURL
			}
			handler.Upstreams[name] = upstream
		default:
			zap.L().Named("config").Fatal("unknown upstream type", zap.String("type", upstreamConfig["type"]))
		}
	}

	// rules
	for _, rule := range configMap.Rules {
		handler.AddRule(rule)
	}

	// listen
	var servers []Server
	listen := configMap.Listen
	for _, serverConfig := range listen {
		checkMapAttrs(serverConfig, "listen", "type", "address")
		switch serverConfig["type"] {
		case "udp":
			server := &UDPServer{
				ServerImpl{
					address: serverConfig["address"],
					handler: handler,
				},
			}
			servers = append(servers, server)
		case "tcp":
			server := &TCPServer{
				ServerImpl{
					address: serverConfig["address"],
					handler: handler,
				},
			}
			servers = append(servers, server)
		default:
			zap.L().Named("config").Fatal("unknown listen type", zap.String("type", serverConfig["type"]))
		}
	}

	zap.L().Named("config").Info("config file read",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("servers", len(servers)),
		zap.Int("upstreams", len(configMap.Upstreams)),
		zap.Int("rules", len(configMap.Rules)),
	)

	return servers
}
