# syntax=docker/dockerfile:1
FROM golang:1.23.2 AS build

ARG BUILDTIME
ARG REVISION
ARG VERSION
ARG GOCACHE
ARG GOMODCACHE

RUN mkdir -p /src
COPY . /src/
WORKDIR /src

# Download taskfile to run the build command
RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin

# Cache go mod dependencies to speed up repeated builds
RUN --mount=type=cache,id=${GOCACHE},target=/root/.cache/go-build \
    --mount=type=cache,id=${GOMODCACHE},target=/go/pkg/mod \
    task build \
    VERSION=${VERSION} REVISION=${REVISION} BUILDTIME=${BUILDTIME}

FROM scratch

WORKDIR /app
COPY --from=build /src/bin/capargo /app/capargo
USER 1001:1001
ENTRYPOINT ["/app/capargo"]
