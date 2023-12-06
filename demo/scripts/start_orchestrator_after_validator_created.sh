#!/bin/bash

# This script waits  for the validator to be created before starting the orchestrator

# check if environment variables are set
if [[ -z "${MONIKER}" || -z "${PRIVATE_KEY}" ]] || \
   [[ -z "${CORE_GRPC_HOST}" || -z "${CORE_GRPC_PORT}" ]] || \
   [[ -z "${CORE_RPC_HOST}" || -z "${CORE_RPC_PORT}" ]] || \
   [[ -z "${P2P_LISTEN}" ]]
then
  echo "Environment not setup correctly. Please set:"
  echo "MONIKER, PRIVATE_KEY, CORE_GRPC_HOST, CORE_GRPC_PORT, CORE_RPC_HOST, CORE_RPC_PORT, P2P_LISTEN variables"
  exit 1
fi

# initialize orchestrator
/bin/blobstream orch init

# add keys to keystore
/bin/blobstream orch keys evm import ecdsa "${PRIVATE_KEY}" --evm.passphrase 123

# import the p2p key to use
/bin/blobstream orchestrator keys p2p import key "${P2P_IDENTITY}"

/bin/blobstream orchestrator start \
  --evm.account="${EVM_ACCOUNT}" \
  --core.rpc="${CORE_RPC_HOST}:${CORE_RPC_PORT}" \
  --core.grpc="${CORE_GRPC_HOST}:${CORE_GRPC_PORT}" \
  --grpc.insecure \
  --p2p.nickname=key \
  --p2p.listen-addr="${P2P_LISTEN}" \
  --evm.passphrase=123
