FROM alpine:latest

ADD src /

ENV DOCKER_HOST unix:///tmp/docker.sock

RUN apk update && \
    apk add supervisor haproxy ca-certificates curl bash && \
    rm -rf /var/cache/apk/* && \
    curl -L https://github.com/jwilder/docker-gen/releases/download/0.7.3/docker-gen-linux-amd64-0.7.3.tar.gz | tar -C /usr/local/bin -xvz

ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
