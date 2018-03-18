[![Docker Build Status](https://img.shields.io/docker/build/flavioaiello/swarm-router.svg?style=for-the-badge)](https://hub.docker.com/r/flavioaiello/swarm-router/)
[![Docker Stars](https://img.shields.io/docker/stars/flavioaiello/swarm-router.svg?style=for-the-badge)](https://hub.docker.com/r/flavioaiello/swarm-router/)
[![Docker Pulls](https://img.shields.io/docker/pulls/flavioaiello/swarm-router.svg?style=for-the-badge)](https://hub.docker.com/r/flavioaiello/swarm-router/)

# Swarm-Router
The «true zero config» production ready ingress router for Docker swarm mode deployments, based on the mature and superior haproxy library.

Unique advantages over treafik, gobetween, sniproxy, flow-proxy and many others:
- Zero-copy using the splice syscall allowing real gbps throughput at very low cpu
- No root privileges required
- No socket mount required
- No external dependencies

## Scope
Solves common docker swarm mode large scale requirements:
- Port overlapping on HTTP and TLS when publishing by service FQDN endpoint
- TLS termination optionally with X.509 mutual auth
- End to end encryption with TLS passthrough when using TLS encryption
- Docker swarm mode stack isolation by swarm-router cascading

## Getting Started
Common docker swarm mode platform requirements can be accomplished by combining different swarm-router capabilites.

## Basic configuration
The swarm-router can listen on multiple ports as shown below. Encryption can be optionally activated providing your fullchain certificate. This file should also contain your private key. Preferably this one should be mounted using docker secrets.
```
HTTP_PORTS=80 88 8080
TLS_PORTS=443 8443
TLS_CERT=/data/certs/fullchain.pem
```

### Testdrive
A first testdrive can be made by starting the swarm-router in legacy mode by using `docker run ` as shown below:
```
docker run --name swarm-router -d -e HTTP_PORTS=80 -e TLS_PORTS=443 -p 80:80 -p 443:443 -p 1111:1111 flavioaiello/swarm-router
```

### Docker Swarm Mode ingress routing

```
version

```

### Docker Swarm Mode ingress routing with stack isolation


```
version
```
