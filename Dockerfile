# stage 1 Build qgb binary
FROM golang:1.19-alpine as builder
RUN apk update && apk --no-cache add make gcc musl-dev git
COPY . /orchestrator-relayer
WORKDIR /orchestrator-relayer
RUN make build

# stage 2
FROM alpine:3.17.2
# hadolint ignore=DL3018
RUN apk update && apk --no-cache add bash

COPY --from=builder /orchestrator-relayer/build/qgb /bin/qgb

# p2p port
EXPOSE 30000

ENTRYPOINT [ "/bin/qgb" ]
