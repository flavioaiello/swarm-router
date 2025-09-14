[![Go Report](
https://goreportcard.com/badge/github.com/flavioaiello/swarm-router)](https://goreportcard.com/report/github.com/flavioaiello/swarm-router)


# Swarm-Router

**Swarm-Router** is a straightforward ingress router designed for Docker Swarm mode that requires no manual setup. It leverages HAProxy to automatically discover backend services through DNS and directs traffic using HTTP Host headers or TLS SNI.

## Main Advantages

### Efficiency & Safety
- **Zero-Copy Design**: Uses TCP splicing for high-speed data transfer with low CPU overhead.
- **Secure by Default**: Operates without needing root access or connecting to the Docker socket.
- **Minimal Footprint**: Runs independently without requiring additional software.

### Docker Swarm Integration
- **Resolves Port Issues**: Prevents conflicts when publishing service names.
- **Smart Discovery**: Uses split DNS for automatic backend identification.
- **Versatile Protocol Handling**: Manages both HTTP routing and TLS processing.
- **Adaptable TLS Handling**: Offers options for both TLS termination and passthrough.
- **Scalable Deployment**: Can serve as the main entry point for the whole swarm or for individual stacks.

This tool simplifies Docker Swarm networking by automating routing tasks while ensuring high performance and ease of use.

## Typical Applications

### Incoming Traffic Management
Forward-Haproxy can direct incoming requests based on local DNS entries.

### Distributing Workloads
Ideal for balancing loads in simpler, DNS-based environments.

## Network Compatibility

- **HTTP Routing**: Uses the HTTP Host header for routing decisions.
- **TCP Routing**: Utilizes TLS SNI for connection handling, with optional TLS offloading.
- **Auto Service Detection**: Finds service endpoints dynamically via DNS.

## Getting Started

Deploy the router and sample applications with:

```
docker stack deploy -c swarm.yml demo
```

After deployment, access the demo apps at:

- http://app1.localtest.me
- http://app2.localtest.me

## Container Image
Built from: `haproxy:lts-alpine` (Long Term Support version)

- No root access needed
- No extra dependencies

## Health Monitoring
HAProxy provides a health endpoint on port 1111:

```
wget -qO- http://127.0.0.1:1111/
```

## TLS / Certificate Management
For TLS termination:
Store fullchain PEM files (including private keys) in `/certs/` using the naming convention:

`<service>.com.pem`

Mount these using Docker volumes or secrets.

Optional: Configure HAProxy to add security headers like HSTS for HTTP traffic.

## Performance Characteristics
Employs zero-copy forwarding and TCP splicing for maximum throughput and minimal CPU usage.

Efficient handling of both HTTP and TCP traffic with low latency.

Outputs structured JSON logs for easy integration with logging platforms like ELK or Fluentd.

## Monitoring & Metrics
Status information is available on port 1111.

Prometheus-formatted metrics can be accessed at `/metrics`:
`http://localhost:1111/metrics`

# Licensing
This project is licensed under the MIT License. See the LICENSE file for details.