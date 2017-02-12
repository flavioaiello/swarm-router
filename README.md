# SNI-Router
Very lean dynamic traffic router based on alpine linux and haproxy, optionally with encryption passtrough based on X.509 mutual auth. For a more sophisticated setup you could either derive from this image and extend it with a replicated KVS and Keepalived or use Traefik.io.

## Scope
This docker container is inspired by jwilder's nginx automatic reverse proxy and is using his docker-gen library to generate configuration files up to the actual docker runtime.
It accomplishes the same as the mentioned reverse proxy based on nginx and solves multiple connectivity issues using haproxy instead:
- Port overlapping on HTTP and TCP (eg. SNI on TLS)
- End to end encryption with TLS passthrough (This is the SNI-Router part)
- Automatic reconfiguration when further containers are spinned up or removed
- TLS Offloading with SNI-Routing

## Docker compose sample excerpts

### Standard port routing for http apps

```
version: '2'

services:

    sni-router:
        image: flavioaiello/sni-router
        volumes:
            - /var/run/docker.sock:/tmp/docker.sock
        environment:
            - ROUTER_HTTP_PORT=80
        ports:
            - "80:80"
            - "1111:1111"         # Stats listening on Port 1111
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
        image: flavioaiello/sni-router
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
### Standard port routing for tcp apps (https etc.)
```
version: '2'

services:

    sni-router:
        image: flavioaiello/sni-router
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
### Standard port routing for tcp service (eg. mongodb, etc. )
```
version: '2'

services:

    sni-router:
        image: flavioaiello/sni-router
        volumes:
            - /var/run/docker.sock:/tmp/docker.sock
        environment:
            - ROUTER_TCP_PORT=27017
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
### TLS Offloading with SNI Routing for tcp service (eg. mySQL, etc. )
```
version: '2'

services:

    sni-router:
      image: flavioaiello/sni-router
      volumes:
        - /var/run/docker.sock:/tmp/docker.sock
        - /data/certs/:/certs/:ro
      environment:
        - ROUTER_TCP_PORT=443
        - TLS_CERT=/certs/fullchain.pem
      ports:
        - "443:443"
        - "1111:1111"
      restart: always

    yours_tcp_on_custom_port:
      ...
      environment:
        - VIRTUAL_HOST=db.myfancywebsite.com
        - VIRTUAL_TCP_PORT=3306
      ...
```

## Contribute
If you want to further customize this image, please feel free to contribute.
