FROM alpine:3.7

COPY files /

ENV DOCKER_HOST unix:///tmp/docker.sock

RUN set -ex;\
    apk update;\
    apk upgrade;\
    apk add supervisor haproxy ca-certificates curl bash;\
    rm -rf /var/cache/apk/*;\
    curl -L https://github.com/jwilder/docker-gen/releases/download/0.7.3/docker-gen-linux-amd64-0.7.3.tar.gz | tar -C /usr/local/bin -xvz

ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
