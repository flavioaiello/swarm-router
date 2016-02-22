FROM alpine:latest

ENV DOCKER_HOST unix:///tmp/docker.sock

RUN apk update && \
    apk add supervisor haproxy ca-certificates curl && \
    rm -rf /var/cache/apk/* && \
    curl -L https://github.com/jwilder/docker-gen/releases/download/0.6.0/docker-gen-linux-amd64-0.6.0.tar.gz | tar -C /usr/local/bin -xvz && \
    mkdir -p /var/log/haproxy/

ADD src /

ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
