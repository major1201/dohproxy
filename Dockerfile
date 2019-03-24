FROM alpine:3.9

ENV DOHPROXY_VER "0.2.0"

ADD https://github.com/major1201/dohproxy/releases/download/v${DOHPROXY_VER}/dohproxy-linux_amd64-${DOHPROXY_VER}.tar.gz /tmp/

RUN tar zxf /tmp/dohproxy-linux_amd64-${DOHPROXY_VER}.tar.gz -C /usr/sbin/ \
    && mv /usr/sbin/dohproxy-linux_amd64-${DOHPROXY_VER} /usr/sbin/dohproxy \
    && mkdir -p /etc/dohproxy \
    && apk update \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/*

COPY example-config-docker.yml /etc/dohproxy/dohproxy.yml

VOLUME /etc/dohproxy

EXPOSE 53/udp

CMD ["/usr/sbin/dohproxy", "-c", "/etc/dohproxy/dohproxy.yml"]
