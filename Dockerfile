# syntax=docker/dockerfile:1
ARG GO_VERSION=1.26
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
ARG TARGETOS
ARG TARGETARCH
ARG NEXTTIMER_VERSION=dev
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
# Resolve Bubble Tea dependencies for the standalone terminal client.
RUN go mod tidy
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X github.com/desirevolution/next-timer/internal/server.AppVersion=${NEXTTIMER_VERSION}" -o /out/nexttimer-server ./cmd/nexttimer-server
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags='-s -w' -o /out/nexttimer ./cmd/nexttimer

FROM scratch AS binary
COPY --from=build /out/ /

FROM alpine:3.23 AS runtime
RUN addgroup -S timer && adduser -S timer -G timer && mkdir /data && chown timer:timer /data
COPY --from=build /out/nexttimer-server /usr/local/bin/nexttimer-server
USER timer
VOLUME /data
EXPOSE 8080
ENV TIMER_LISTEN=0.0.0.0:8080 TIMER_DATA=/data/templates.json
ENTRYPOINT ["nexttimer-server"]
