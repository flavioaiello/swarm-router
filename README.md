# SNI-Router

## Scope
This docker container solves multiple connectivity issues when working at large scale with docker:
- Port overlapping on HTTP and TCP (SNI on TLS)
- End to end encryption with TLS passthrough
- Automatic reconfiguration when further containers are spinned up

##  Whats inside
- Alpine linux
- HAProxy 
- Docker-gen

## Usage
Feel free to use docker-compose 1.6 sample excerpts for your specific use case. HAPROXY stats are listening on Port 1111.

## Standard port routing for http apps

```
version: '2'

services:

    sni-router:
        build: serverking/sni-router:latest
        volumes:
            - /var/run/docker.sock:/tmp/docker.sock
        environment:
            - ROUTER_HTTP_PORT=80
        ports:
            - "80:80"
            - "1111:1111"
        restart: always

    yours_http_on_default_port:
        ...
        environment:
            - VIRTUAL_HOST=www.myfancywebsite.com
        ...
```
### Custom port routing for http apps
```
version: '2'

services:

    sni-router:
        build: serverking/sni-router:latest
        volumes:
            - /var/run/docker.sock:/tmp/docker.sock
        environment:
            - ROUTER_HTTP_PORT=80
        ports:
            - "80:80"
            - "1111:1111"
        restart: always

    yours_http_on_custom_port:
        ...
        environment:
            - VIRTUAL_HOST=admin.myfancywebsite.com
            - VIRTUAL_HTTP_PORT=8000
        ...
```
## Standard port routing for tcp apps (https etc.)
```
version: '2'

services:

    sni-router:
        build: serverking/sni-router:latest
        volumes:
            - /var/run/docker.sock:/tmp/docker.sock
        environment:
            - ROUTER_TCP_PORT=443
        ports:
            - "443:443"
            - "1111:1111"
        restart: always

    yours_tcp_on_custom_port:
        ...
        environment:
            - VIRTUAL_HOST=www.myfancywebsite.com
            - VIRTUAL_TCP_PORT=443
        ...
```
## Standard port routing for tcp service (eg. mongodb, etc. )
```
version: '2'

services:

    sni-router:
        build: serverking/sni-router:latest
        volumes:
            - /var/run/docker.sock:/tmp/docker.sock
        environment:
            - ROUTER_TCP_PORT=443
        ports:
            - "27017:27017"
            - "1111:1111"
        restart: always

    yours_tcp_on_custom_port:
        ...
        environment:
            - VIRTUAL_HOST=mongodb.myfancywebsite.com
            - VIRTUAL_TCP_PORT=27017
        ...
```

## Todo's
- [x] Review supervisor configuration (reload and restart on new containers works now)
- [ ] Logging to stdout and stderr
