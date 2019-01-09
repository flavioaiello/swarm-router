[![Docker Build Status](https://img.shields.io/docker/build/flavioaiello/swarm-router.svg)](https://hub.docker.com/r/flavioaiello/swarm-router/)
[![Docker Stars](https://img.shields.io/docker/stars/flavioaiello/swarm-router.svg)](https://hub.docker.com/r/flavioaiello/swarm-router/)
[![Docker Pulls](https://img.shields.io/docker/pulls/flavioaiello/swarm-router.svg)](https://hub.docker.com/r/flavioaiello/swarm-router/)
[![Docker Automation](
https://img.shields.io/docker/automated/flavioaiello/swarm-router.svg)](https://hub.docker.com/r/flavioaiello/swarm-router/)
[![Go Report](
https://goreportcard.com/badge/github.com/flavioaiello/swarm-router)](https://goreportcard.com/report/github.com/flavioaiello/swarm-router)

# Swarm-Router
This is the «zero config» ingress router for Docker swarm mode deployments, based on the mature and superior haproxy library and a little of golang offering unique advantages:
- Zero-copy using tcp splice syscall for real gbps throughput at very low cpu
- No root privileges required
- No docker socket mount required for service discovery
- No external dependencies

## Scope
Solves common docker swarm mode requirements:
- Port overlapping due to service name publishing 
- Claim based service discovery
- HTTP service forwarding
- TLS service offloading eg. termination and forwarding
- TLS service passthrough
- Stackable as swarm or stack edge

## Docker Swarm
Built for docker swarm mode ingress networking: Secure service discovery based on url claim nameserver resolution. Just define your service name urls as network alias names.

## Ingress routing
Simply get started by executing `docker stack deploy -c swarm.yml swarm` to have a swarm-router (and portainer for your convenience) up and running. Services can simply be exposed by network alias urls in case they listen on port `8080`. Port override can be done by `HTTP_BACKENDS_PORT` eg. `TLS_BACKENDS_PORT` by `startswith:<port>` pattern.

```
  swarm-router:
    image: swarm-router:latest
    ports:
      - "80:80"
      - "8080:8080"
      - "443:443"
      - "8443:8443"
      - "1111:1111"
    networks:
      default:
      routing:
...
  service:
...
    networks:
      default:
        aliases:
          - myservice.localtest.me
```

### Ingress routing with isolated stacks
Stack isolation when deploying multiple stacks is accomplished by stack edge routers so that service name collissions can be avoided.

![Stack isolation](https://github.com/flavioaiello/swarm-router/blob/master/swarm-router.png?raw=true)

```
  stack-router:
    image: swarm-router:latest
    Environment:
      - FQDN_BACKENDS_HOSTNAME=false
    ports:
      - "8080:8080"
    networks:
      default:
      routing:
       - stack-router.mystack.localtest.me
       - myfirstservice.mystack.localtest.me
       - mysecondservice.mystack.localtest.me
...
  myfirstservice:
    image: ...
...
  mysecondservice:
    image: ...
...
```

Simply deploy the stacks below to have a two sample stacks and swarm-router (and portainer for your convenience as well) up and running.
```
docker stack deploy -c swarm.yml swarm
docker stack deploy -c stack-a.yml astack
docker stack deploy -c stack-b.yml bstack
```

## Certificates
When TLS offloading comes into action, according fullchain certificates containing the private key should be provisioned on `/certs` host volume mount as `service.com.pem`. Preferably this one should be mounted using docker secrets.

## Performance
This one is built for high throughput and little CPU usage. Haproxy implements zero-copy and tcp-splicing based TCP handling. Golang based projects are lacking on those feature support: https://github.com/golang/go/issues/10948. (All golang based projects like Traefik etc. are also affected)

#### Todos
- [ ] add termination with ACME autocerts
