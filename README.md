Static HTTP File Server
=======================

A very small HTTP file server in a container-image, with slightly better security than [http.Dir](https://pkg.go.dev/net/http#Dir)

## Features

- Serve files from the specified directory to HTTP clients.
- Doesn't serve files outside of that directory to HTTP clients.
- Follows symlinks within the specified directory.
- Doesn't follow symlinks that refer to paths outside of the specified directory.
- Attempts to serve `index.html` from directories matched by HTTP requests.

## Usage

```
$ podman run -it --rm --volume your/stuff:/www:ro --publish 9807:9807 ghcr.io/jmanero/static:latest /www
```
