# Use the maintained LTS branch instead of floating alpine
FROM haproxy:lts-alpine

# Copy configuration files
COPY files /
