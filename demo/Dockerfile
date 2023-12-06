FROM ghcr.io/celestiaorg/orchestrator-relayer:v1.0.1 as orchestrator-relayer

FROM ghcr.io/celestiaorg/celestia-app:v1.3.0 as celestia-app

USER root

RUN apk update && apk --no-cache add bash jq coreutils curl

COPY --from=orchestrator-relayer /bin/blobstream /bin/blobstream

# p2p port
EXPOSE 9090 26657 30000

CMD [ "/bin/blobstream" ]
