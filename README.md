# dohproxy

A DNS-over-Https proxy and router

[![GoDoc](https://godoc.org/github.com/major1201/dohproxy?status.svg)](https://godoc.org/github.com/major1201/dohproxy)
[![Go Report Card](https://goreportcard.com/badge/github.com/major1201/dohproxy)](https://goreportcard.com/report/github.com/major1201/dohproxy)

## Installation

```bash
$ go install github.com/major1201/dohproxy
```

## Usage

Start dohproxy with config `/etc/dohproxy.yml`

```
dohproxy
```

Start dohproxy with a custom config path

```bash
dohproxy -c /home/major1201/my-doh-config.yml
```

Service

```bash
# install as a service
dohproxy -c /home/major1201/my-doh-config.yml --service install

# start the service
dohproxy --service start

# stop the service
dohproxy --service stop

# uninstall the service
dohproxy --service uninstall
```

## Configuration

```yml
log:
  stdout: stdout                  # default: stdout, log-to-file on Windows is not supported
  stderr: /var/log/dohproxy.err   # default: stderr, log-to-file on Windows is not supported
  level: info                     # default: debug, choices: debug, info, warn(warning), error, dpanic, panic, fatal

listen:
  - type: udp
    address: 127.0.0.1:53
  - type: tcp
    address: 127.0.0.1:53

upstreams:
  google-public:
    type: dns
    address: 8.8.8.8:53
  my-corp-dns:
    type: dns
    address: 192.168.53.1:53
  doh-get-with-proxy:
    type: doh-get
    address: https://some-doh-server-i-cant.com/dns-query
    proxy: socks5://127.0.0.1:1080
  doh-post:
    type: doh-post
    address: https://cloudflare-dns.com/dns-query

rules:
  - fqdn:cloudflare-dns.com      google-public
  - fqdn:www.my-dev-server.com   10.0.31.1
  - keyword:mycorp.com           my-corp-dns
  - suffix:mybiz.com             my-corp-dns
  - suffix:never-response.com    blackhole
  - suffix:adxxx.com             reject
  - wildcard:*                   doh-post
```

listen types:

- udp
- tcp

upstream types:

- dns: classic DNS server
- doh / doh-get: DNS-over-HTTPS protocol, using HTTP GET method
- doh-post: DNS-over-HTTPS protocol, using HTTP POST method

rule format: `[fqdn|prefix|suffix|keyword|wildcard|regex]:expression upstream|blackhole|reject|static_ip`

- upstream: upstream name defined in the `upstreams` field
- blackhole: it never response to any dns requests, it just does nothing
- reject: returns error immediately

## Known Issues

- The `log.stdout` and `log.stderr` part in config file only support `stdout` on Windows platform, due to `zap` package limit

## Contributing

Just fork the repository and open a pull request with your changes.

## Licence

MIT
