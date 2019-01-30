package main

import (
	"github.com/go-yaml/yaml"
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	Log       *LogConfig
	Listen    []map[string]string
	Upstreams map[string]map[string]string
	Rules     []string
}

// LogConfig describes the log config structure
type LogConfig struct {
	Stdout string
	Stderr string
	Level  string
}

func checkMapAttrs(m map[string]string, parentKey string, keys ...string) {
	for _, key := range keys {
		if _, ok := m[key]; !ok {
			zap.L().Named("config").Fatal("lost key", zap.String("field", parentKey), zap.String("lost key", key))
		}
	}

}

func reloadLogConfig(logConfig *LogConfig) {
	if logConfig == nil {
		return
	}

	stdout := "stdout"
	stderr := "stderr"
	level := zapcore.DebugLevel

	if logConfig.Stdout != "" {
		stdout = logConfig.Stdout
	}
	if logConfig.Stderr != "" {
		stderr = logConfig.Stderr
	}
	if logConfig.Level != "" {
		logLevelMap := map[string]zapcore.Level{
			"debug":   zapcore.DebugLevel,
			"info":    zapcore.InfoLevel,
			"warn":    zapcore.WarnLevel,
			"warning": zapcore.WarnLevel,
			"error":   zapcore.ErrorLevel,
			"dpanic":  zapcore.DPanicLevel,
			"panic":   zapcore.PanicLevel,
			"fatal":   zapcore.FatalLevel,
		}
		if logLevel, ok := logLevelMap[logConfig.Level]; ok {
			level = logLevel
		} else {
			zap.L().Named("config").Fatal("unknown log level", zap.String("log.level", logConfig.Level))
		}
	}

	zap.L().Info("log config reloading", zap.String("stdout", stdout), zap.String("stderr", stderr), zap.Int("level", int(level)))
	initLog(stdout, stderr, level)
}

// LoadServersFromConfig loads the config file in YAML format into Server slice objects
func LoadServersFromConfig(configPath string) []Server {
	logger := zap.L().Named("config")

	logger.Info("reading config file", zap.String("filename", configPath))
	startTime := time.Now()

	yamlFile, err := os.Open(configPath)
	defer yamlFile.Close()
	if err != nil {
		logger.Fatal("can't open config file", zap.String("filename", configPath))
	}

	configMap := &Config{}
	yaml.NewDecoder(yamlFile).Decode(configMap)

	// log
	reloadLogConfig(configMap.Log)

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
					logger.Fatal("upstream proxy url parse error", zap.String("upstream name", name), zap.String("proxy", proxyStr))
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
					logger.Fatal("upstream proxy url parse error", zap.String("upstream name", name), zap.String("proxy", proxyStr))
				}
				upstream.proxy = proxyURL
			}
			handler.Upstreams[name] = upstream
		default:
			logger.Fatal("unknown upstream type", zap.String("type", upstreamConfig["type"]))
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
			logger.Fatal("unknown listen type", zap.String("type", serverConfig["type"]))
		}
	}

	logger.Info("config file read",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("servers", len(servers)),
		zap.Int("upstreams", len(configMap.Upstreams)),
		zap.Int("rules", len(configMap.Rules)),
	)

	return servers
}
