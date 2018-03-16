[![Docker Build Status](https://img.shields.io/docker/build/flavioaiello/swarm-router.svg?style=for-the-badge)](https://hub.docker.com/r/flavioaiello/swarm-router/)
[![Docker Stars](https://img.shields.io/docker/stars/flavioaiello/swarm-router.svg?style=for-the-badge)](https://hub.docker.com/r/flavioaiello/swarm-router/)
[![Docker Pulls](https://img.shields.io/docker/pulls/flavioaiello/swarm-router.svg?style=for-the-badge)](https://hub.docker.com/r/flavioaiello/swarm-router/)

# Swarm-Router
True zero config docker swarm mode traffic router still based on haproxy. Does not need root privileges and even not listening on swarm events. 
Very lean dynamic traffic router based on alpine linux optionally with encryption passtrough based on X.509 mutual auth. 

## Scope
This docker container is inspired by jwilder's nginx automatic reverse proxy and is using his docker-gen library to generate configuration files up to the actual docker runtime.
It accomplishes the same as the mentioned reverse proxy based on nginx and solves multiple connectivity issues using haproxy instead:
- Port overlapping on HTTP and TCP (eg. SNI on TLS)
- End to end encryption with TLS passthrough (This is the SNI-Router part)
- Automatic reconfiguration when further containers are spinned up or removed
- TLS Offloading with SNI-Routing

## Docker stack deploy sample excerpts «.yml»

### Standard port routing for http apps

```
WIP!


```

