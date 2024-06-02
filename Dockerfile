# stage 1 Build blobstream binary
FROM --platform=$BUILDPLATFORM docker.io/golang:1.21.6-alpine3.18 as builder

ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN apk update && apk --no-cache add make gcc musl-dev git bash

COPY . /orchestrator-relayer
WORKDIR /orchestrator-relayer
RUN uname -a &&\
    CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    make build

# final image
FROM docker.io/alpine:3.19.1

ARG UID=10001
ARG USER_NAME=celestia

ENV CELESTIA_HOME=/home/${USER_NAME}

# hadolint ignore=DL3018
RUN apk update && apk add --no-cache \
        bash \
        curl \
        jq \
    # Creates a user with $UID and $GID=$UID
    && adduser ${USER_NAME} \
        -D \
        -g ${USER_NAME} \
        -h ${CELESTIA_HOME} \
        -s /sbin/nologin \
        -u ${UID}

COPY --from=builder /orchestrator-relayer/build/blobstream /bin/blobstream
COPY --chown=${USER_NAME}:${USER_NAME} docker/entrypoint.sh /opt/entrypoint.sh

USER ${USER_NAME}

# p2p port
EXPOSE 30000

ENTRYPOINT [ "/bin/bash", "/opt/entrypoint.sh" ]
