### source build ###
FROM golang:1.11-alpine3.8 as build

COPY src /src

WORKDIR /src

RUN set -ex ;\
    go get -d -v -t;\
    CGO_ENABLED=0 GOOS=linux go build -v -o /files/usr/local/bin/swarm-router

### runtime build ###
FROM haproxy:1.8.16-alpine

COPY files /
COPY --from=build /files /

EXPOSE 80 443 1111

RUN sed -r 's/(exec).+("\$@")/\1 swarm-router \2/g' -i docker-entrypoint.sh
