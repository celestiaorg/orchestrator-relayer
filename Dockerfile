# stage 1 Build qgb binary
FROM golang:1.20-alpine as builder
RUN apk update && apk --no-cache add make gcc musl-dev git
COPY . /orchestrator-relayer
WORKDIR /orchestrator-relayer
RUN make build

# final image
FROM alpine:3.18.2
# hadolint ignore=DL3018
RUN apk update && apk --no-cache add bash

COPY --from=builder /orchestrator-relayer/build/qgb /bin/qgb
COPY --chown=${USER_NAME}:${USER_NAME} docker/entrypoint.sh /opt/entrypoint.sh

# p2p port
EXPOSE 30000

ENTRYPOINT [ "/bin/bash", "/opt/entrypoint.sh" ]
