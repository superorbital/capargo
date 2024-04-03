# syntax=docker/dockerfile:1
FROM golang:1.22.1 as build

ARG BUILDTIME
ARG REVISION
ARG VERSION

# Download taskfile to run the build command
RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin

RUN mkdir -p /src
COPY . /src/
WORKDIR /src

# Cache go mod dependencies to speed up repeated builds
RUN --mount=type=cache,id=${GOCACHE},target=/root/.cache/go-build \
    --mount=type=cache,id=${GOMODCACHE},target=/go/pkg/mod \
    VERSION=${VERSION} REVISION=${REVISION} BUILDTIME=${BUILDTIME} \
    task build

FROM scratch

WORKDIR /app
COPY --from=build /src/bin/capargo /app/capargo
ENTRYPOINT ["/app/capargo"]
