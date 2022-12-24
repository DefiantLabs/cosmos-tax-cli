FROM golang:1.19-alpine3.16 AS build-env

# Customize to your build env
ARG ARCH
ARG TARGETARCH
ARG TARGETOS=linux
ARG BUILD_TAGS=muslc
ARG LD_FLAGS=-linkmode=external -extldflags '-Wl,-z,muldefs -static'

# Customise to your repo.
ARG GITHUB_ORGANIZATION=DefiantLabs
ARG REPO_HOST=github.com
ARG GITHUB_REPO=cosmos-tax-cli-private
ARG VERSION=latest

# Install cli tools for building and final image
RUN apk add --update --no-cache curl make git libc-dev bash gcc linux-headers eudev-dev ncurses-dev libc6-compat jq

# Copy files required for building
WORKDIR /go/src/${REPO_HOST}/${GITHUB_ORGANIZATION}/${GITHUB_REPO}
COPY . .

# Install build dependencies.

ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.1.1/libwasmvm_muslc.${ARCH}.a /lib/libwasmvm_muslc.${ARCH}.a

RUN sha256sum /lib/libwasmvm_muslc.${ARCH}.a
RUN cp /lib/libwasmvm_muslc.${ARCH}.a /lib/libwasmvm_muslc.a
# RUN cp /lib64/ld-linux-x86-64.so.2 /lib64/libdl.so.2

# Build the app
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=1 go install -ldflags ${LD_FLAGS} -tags ${BUILD_TAGS}

# Build a sub app
WORKDIR /go/src/${REPO_HOST}/${GITHUB_ORGANIZATION}/${GITHUB_REPO}/client
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=1 go install -ldflags ${LD_FLAGS} -tags ${BUILD_TAGS}

# Use busybox to create a user
FROM busybox:stable-musl AS busybox
RUN addgroup --gid 1137 -S defiant && adduser --uid 1137 -S defiant -G defiant

# Use scratch for the final image
FROM scratch
ARG ARCH=x86_64

# Label should match your github repo
LABEL org.opencontainers.image.source="https://github.com/defiantlabs/cosmos-tax-cli-private"

# Install Binaries
COPY --from=build-env /go/bin /bin
COPY --from=build-env /usr/bin/ldd /bin/ldd
COPY --from=build-env /usr/bin/curl /bin/curl
COPY --from=build-env /usr/bin/jq /bin/jq

# Install Libraries
COPY --from=build-env /usr/lib/libgcc_s.so.1 /lib/
COPY --from=build-env /lib/ld-musl-${ARCH}.so.1 /lib
COPY --from=build-env /usr/lib/libonig.so.5 /lib
COPY --from=build-env /usr/lib/libcurl.so.4 /lib
COPY --from=build-env /lib/libz.so.1 /lib
COPY --from=build-env /usr/lib/libnghttp2.so.14 /lib
COPY --from=build-env /lib/libssl.so.1.1 /lib
COPY --from=build-env /lib/libcrypto.so.1.1 /lib
COPY --from=build-env /usr/lib/libbrotlidec.so.1 /lib
COPY --from=build-env /usr/lib/libbrotlicommon.so.1 /lib

# Install trusted CA certificates
COPY --from=build-env /etc/ssl/cert.pem /etc/ssl/cert.pem

# Install cli tools from busybox
COPY --from=busybox /bin/ln /bin/ln
COPY --from=busybox /bin/cp /bin/cp
COPY --from=busybox /bin/ls /bin/ls
COPY --from=busybox /bin/busybox /bin/sh
COPY --from=busybox /bin/cat /bin/cat
COPY --from=busybox /bin/less /bin/less
COPY --from=busybox /bin/grep /bin/grep
COPY --from=busybox /bin/sleep /bin/sleep
COPY --from=busybox /bin/env /bin/env
COPY --from=busybox /bin/tar /bin/tar
COPY --from=busybox /bin/tee /bin/tee
COPY --from=busybox /bin/du /bin/du
COPY --from=busybox /bin/df /bin/df
COPY --from=busybox /bin/nc /bin/nc
COPY --from=busybox /bin/netstat /bin/netstat

# Copy user from busybox to scratch
COPY --from=busybox /etc/passwd /etc/passwd
COPY --from=busybox --chown=1137:1137 /home/defiant /home/defiant

# Set home directory and user
WORKDIR /home/defiant
USER defiant
