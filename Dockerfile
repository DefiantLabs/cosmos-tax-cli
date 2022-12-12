# FROM golang:1.19-alpine AS build-env
FROM ubuntu AS build-env
LABEL org.opencontainers.image.source="https://github.com/defiantlabs/cosmos-tax-cli-private"

ENV PACKAGES make git ssh gcc musl-dev golang net-tools curl

# RUN apk add --no-cache $PACKAGES
RUN apt-get -y update && apt-get install -y $PACKAGES

# Copy the App
WORKDIR /go/src/github.com/DefiantLabs/cosmos-tax-cli-private/cosmos-tax-cli-private
ADD . .

# Build Defaults
ARG TARGETARCH=amd64
ARG TARGETOS=linux

# Build Local App.
RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install -ldflags="-w -s"

# Move all binaries to path
RUN cp /root/go/bin/* /bin/

# Clean up to make image smaller.
RUN rm -rf /go/src/
RUN apt-get clean autoclean
RUN apt-get autoremove --yes
RUN rm -rf /var/lib/{apt,dpkg,cache,log}/