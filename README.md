[![Docker Build Status](https://img.shields.io/docker/build/flavioaiello/swarm-router.svg)](https://hub.docker.com/r/flavioaiello/swarm-router/)
[![Docker Stars](https://img.shields.io/docker/stars/flavioaiello/swarm-router.svg)](https://hub.docker.com/r/flavioaiello/swarm-router/)
[![Docker Pulls](https://img.shields.io/docker/pulls/flavioaiello/swarm-router.svg)](https://hub.docker.com/r/flavioaiello/swarm-router/)
# Swarm-Router
The «zero config» ingress router for Docker swarm mode deployments, based on the mature and superior haproxy library and a little of golang offering unique advantages:

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

## Docker Swarm Mode 1.12+
Built for docker swarm mode ingress networking: Secure service discovery using fqdn forwarding with dns resolution based on  embedded dns. Therefore there is no need to mount the docker socket and maintain labels on compose recipe. Just define your fully qualified service names per network as shown in the sample excerpts below.

## Performance
This one is built for high throughput and little CPU usage. Haproxy implements zero-copy and tcp-splicing based TCP handling. Golang based projects are lacking on those feature support: https://github.com/golang/go/issues/10948. (All golang based projects like Traefik etc. are also affected)

## Getting started
Common docker swarm mode requirements can be accomplished by combining different swarm-router capabilites. The main use cases are made by single swarm-router instance or in cascading mode to isolate stacks from each other.

### Ingress routing
The simplest use case is to spin-up one swarm-router per docker swarm mode cluster.

Execute `docker stack deploy -c swarm.yml swarm` to have a swarm-router (and portainer for your convenience) up and running. Your services can be exposed by simply add a network alias name in case they are listening on port 8080. In any other case you can eighter switch the default port or override based on a namepatter like startswith:<port>.

```
...
  myservice:
...
    networks:
      default:
        aliases:
          - myservices.vcap.me
```

### Ingress routing with stack isolation
Providing an additional swarm-router offers isolation, thus the ability to deploy the same stack with different service names multiple times:
```
docker stack deploy -c swarm.yml swarm
docker stack deploy -c stack-a.yml astack
docker stack deploy -c stack-b.yml bstack
```
![Stack isolation](https://github.com/flavioaiello/swarm-router/blob/master/swarm-router.png?raw=true

Your services can be exposed by simply adding network alias names in case they are listening on port 8080 on stack level swarm-router. In any other case you can eighter switch the default port or override based on a name pattern like startswith:<port>.
For intrastack communication you can either use the service short names or define more short named aliases.

## Configuration

### Listeners

HTTP and TLS ports describe a set of listening sockets accepting client connections.
```
HTTP_PORTS=80 88 8080
TLS_PORTS=443 8443
```

### Backends

#### Default ports
The default port for all backends which the router will connect and forward incoming connections.
```
HTTP_BACKENDS_DEFAULT_PORT=8080
TLS_BACKENDS_DEFAULT_PORT=8443
```
#### Specific ports
Additional port for backends which will partly match the FQDN the router will connect and forward incoming connections.
```
HTTP_BACKENDS_PORT=<value> (optional: startswith;9000 startswithsomethigelse;9090)
TLS_BACKENDS_PORT=<value> (optional: startswith;9000 startswithsomethigelse;9090)
```
#### Todos
- [ ] add ttl to backends

#### Insights
If no backends are known to handle the request, but the FQDN is propagated by swarm, the connection will be forwarded to the swarm-router service listeners. The swarm-router default listeners do NO need any further configuration and will propagated to the default haproxy.tmpl configuration file.
```
HTTP_SWARM_ROUTER_PORT=10080 (default)
TLS_SWARM_ROUTER_PORT=10443 (default)
```
### TLS Encryption
#### Termination
Encryption can be optionally activated providing your fullchain certificate. This file should also contain your private key. Preferably this one should be mounted using docker secrets.
```
TLS_CERT=/data/certs/fullchain.pem
```
#### Todos
- [ ] add tls (sni) passthrough
- [ ] add termination with ACME autocerts
