FROM docker.io/golang:1.18 AS build

RUN mkdir p /build
WORKDIR /build

COPY pkg ./pkg
COPY go.mod go.sum main.go ./

RUN go mod download
RUN go build -o static .

FROM registry.fedoraproject.org/fedora-minimal:36 AS mime
RUN microdnf install -y shared-mime-info

FROM ghcr.io/jmanero/stock:go-fc36

## Get builtin mime database from fc36
COPY --from=mime /usr/share/mime/globs2 /usr/share/mime/globs2
COPY --from=build /build/static /usr/bin/static

ENTRYPOINT [ "/usr/bin/static" ]
