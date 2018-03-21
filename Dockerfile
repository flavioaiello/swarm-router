### source build ###
FROM golang:1.9 as build

COPY build /

WORKDIR /src

RUN set -ex ;\
    go get -d -v -t;\
    CGO_ENABLED=0 GOOS=linux go build -v -o /files/usr/local/bin/swarm-router

### runtime build ###
FROM haproxy:alpine

COPY files /
COPY --from=build /files /

RUN set -ex ;\
    apk update ;\
    apk upgrade ;\
    rm -rf /var/cache/apk/*

EXPOSE 80 443 1111

CMD ["/usr/local/bin/swarm-router"]
