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

# wait for the validator to be created before starting the orchestrator
VAL_ADDRESS=$(celestia-appd keys show ${MONIKER} --keyring-backend test --bech=val --home /opt -a)
while true
do
  # verify that the node is listening on gRPC
  nc -z -w5 $CORE_GRPC_HOST $CORE_GRPC_PORT
  result=$?
  if [ "${result}" != "0" ]; then
    echo "Waiting for node gRPC to be available ..."
    sleep 1s
    continue
  fi

  # verify if RPC is running and the validator was created
  output=$(celestia-appd query staking validator ${VAL_ADDRESS} --node tcp://$CORE_RPC_HOST:$CORE_RPC_PORT 2>/dev/null)
  if [[ -n "${output}" ]] ; then
    break
  fi
  echo "Waiting for validator to be created..."
  sleep 3s
done

# initialize orchestrator
/bin/blobstream orch init

# add keys to keystore
/bin/blobstream orch keys evm import ecdsa "${PRIVATE_KEY}" --evm.passphrase 123

# start orchestrator
if [[ -z "${P2P_BOOTSTRAPPERS}" ]]
then
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
else
  # to give time for the bootstrappers to be up
  sleep 5s

  /bin/blobstream orchestrator start \
    --evm.account="${EVM_ACCOUNT}" \
    --core.rpc="${CORE_RPC_HOST}:${CORE_RPC_PORT}" \
    --core.grpc="${CORE_GRPC_HOST}:${CORE_GRPC_PORT}" \
    --grpc.insecure \
    --p2p.listen-addr="${P2P_LISTEN}" \
    --p2p.bootstrappers="${P2P_BOOTSTRAPPERS}" \
    --evm.passphrase=123
fi
