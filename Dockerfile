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
RUN CGO_ENABLED=1 LDFLAGS='-linkmode external -extldflags "-static"' GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install
RUN wget -O /lib/libwasmvm.x86_64.so https://github.com/CosmWasm/wasmvm/raw/main/internal/api/libwasmvm.x86_64.so

# Move all binaries to path
RUN cp /root/go/bin/* /bin/

# Defaults
ARG USERNAME=defiant
ARG USER_UID=1137
ARG USER_GID=$USER_UID

# Create the user
RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && apt-get update \
    && apt-get install -y sudo \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

# Clean up to make image smaller.
RUN rm -rf /go/src/
RUN apt-get clean autoclean
RUN apt-get autoremove --yes
RUN rm -rf /var/lib/{apt,dpkg,cache,log}/

USER $USERNAME
WORKDIR /home/defiant
