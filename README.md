# SNI-Router
Very lean dynamic traffic router based on alpine linux and haproxy, optionally with encryption passtrough based on X.509 mutual auth. For a more sophisticated setup you could either derive from this image and extend it with a replicated KVS and Keepalived or use Traefik.io.

## Scope
This docker container is inspired by jwilder's nginx automatic reverse proxy and is using his docker-gen library to generate configuration files up to the actual docker runtime.
It accomplishes the same as the mentioned reverse proxy based on nginx and solves multiple connectivity issues using haproxy instead:
- Port overlapping on HTTP and TCP (eg. SNI on TLS)
- End to end encryption with TLS passthrough (This is the SNI-Router part)
- Automatic reconfiguration when further containers are spinned up or removed

## Docker compose sample excerpts

### Standard port routing for http apps

```
version: '2'

services:

    sni-router:
        build: .
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
        build: .
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
        build: .
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
        build: .
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

## Versioning
Versioning is an issue when deploying the latest release. For this purpose an additional label will be provided during build time. 
The Dockerfile must be extended with an according label argument as shown below:
```
ARG TAG
LABEL TAG=${TAG}
```
Arguments must be passed to the build process using `--build-arg TAG="${TAG}"`.

## Reporting
```
docker inspect --format \
&quot;{{ index .Config.Labels \&quot;com.docker.compose.project\&quot;}},\
 {{ index .Config.Labels \&quot;com.docker.compose.service\&quot;}},\
 {{ index .Config.Labels \&quot;TAG\&quot;}},\
 {{ index .State.Status }},\
 {{ printf \&quot;%.16s\&quot; .Created }},\
 {{ printf \&quot;%.16s\&quot; .State.StartedAt }},\
 {{ index .RestartCount }}&quot; $(docker ps -f name=${STAGE} -q) &gt;&gt; reports/${SHORTNAME}.report
```

