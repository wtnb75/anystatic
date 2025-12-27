# Anystatic

Anystatic is a small, efficient static file server written in Go. It prefers pre-compressed files (for example `.gz` or `.br`) when the client's `Accept-Encoding` header indicates support for the corresponding encoding, similar to nginx's `gzip_static` feature. It can run standalone or as a Traefik plugin to add the same behavior to Traefik routes.

## Features

- Serve pre-compressed files when available
    - Supported encodings: gzip, brotli, zstd
    - No inline compression needed, saving CPU
- Set appropriate `Content-Encoding` and `Vary: Accept-Encoding` headers
- Works as a Traefik plugin or as a standalone HTTP server

## Quick Start

### Install (standalone)

```bash
go install github.com/wtnb75/anystatic/cmd/anystatic@latest
```

### Run (standalone)

```bash
$(go env GOPATH)/bin/anystatic -dir=/var/www -listen=:8080
```

## Using as a Traefik Plugin

When used as a Traefik plugin, Anystatic serves pre-compressed files when the request's `Accept-Encoding` header matches an available compressed variant.

Example configuration (YAML):

```yaml
experimental:
  plugins:
    anystatic:
      modulename: github.com/wtnb75/anystatic
      version: 1.0.0
http:
  routers:
    my-router:
      rule: Host(`hostname.localhost`)
      middlewares: my-static
  middlewares:
    my-static:
      plugin:
        anystatic:
          rootdir: /var/www
```

see also: [compose.yml](./compose.yml)

## Expected HTTP Behavior

- When a client sends `Accept-Encoding: gzip` and a compressed file `path/to/file.gz` exists, the server returns that file with the `Content-Encoding: gzip` header.
- The server sets `Vary: Accept-Encoding` on responses that may differ based on the client's encoding preferences.
- If no compressed variant matches the client's accepted encodings, the server falls back to the uncompressed file.
