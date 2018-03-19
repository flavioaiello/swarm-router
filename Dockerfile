### Source build ###
FROM golang:1.9 as build

COPY src /src

WORKDIR /src

RUN go get -d -v -t;\
    CGO_ENABLED=0 GOOS=linux go build -v -o /usr/local/bin/swarm-router

### Runtime build ###
FROM alpine:edge

COPY files /
COPY --from=build /usr/local/bin/swarm-router /usr/local/bin/

RUN set -ex;\
    apk update;\
    apk upgrade;\
    apk add supervisor haproxy ca-certificates;\
    rm -rf /var/cache/apk/*

EXPOSE 80 443 1111

ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
#CMD ["/usr/local/bin/swarm-router"]
