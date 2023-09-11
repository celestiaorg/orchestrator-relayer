# stage 1 Build qgb binary
FROM --platform=$BUILDPLATFORM docker.io/golang:1.21-alpine3.18 as builder
RUN apk update && apk --no-cache add make gcc musl-dev git
COPY . /orchestrator-relayer
WORKDIR /orchestrator-relayer
RUN make build

# final image
FROM --platform=$BUILDPLATFORM docker.io/alpine:3.18.3

ARG UID=10001
ARG USER_NAME=celestia

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

COPY --from=builder /orchestrator-relayer/build/qgb /bin/qgb
COPY --chown=${USER_NAME}:${USER_NAME} docker/entrypoint.sh /opt/entrypoint.sh

USER ${USER_NAME}

# p2p port
EXPOSE 30000

ENTRYPOINT [ "/bin/bash", "/opt/entrypoint.sh" ]
