#!/bin/bash

# This script runs the QGB relayer with the ability to deploy a new QGB contract or
# pass one as an environment variable QGB_CONTRACT

# check if environment variables are set
if [[ -z "${EVM_CHAIN_ID}" || -z "${PRIVATE_KEY}" ]] || \
   [[ -z "${CORE_GRPC_HOST}" || -z "${CORE_GRPC_PORT}" ]] || \
   [[ -z "${CORE_RPC_HOST}" || -z "${CORE_RPC_PORT}" ]] || \
   [[ -z "${EVM_ENDPOINT}" || -z "${P2P_BOOTSTRAPPERS}" ]] || \
   [[ -z "${P2P_LISTEN}" ]]
then
  echo "Environment not setup correctly. Please set:"
  echo "EVM_CHAIN_ID, PRIVATE_KEY, CORE_GRPC_HOST, CORE_GRPC_PORT, CORE_RPC_HOST, CORE_RPC_PORT, EVM_ENDPOINT, P2P_BOOTSTRAPPERS, P2P_LISTEN variables"
  exit 1
fi

# wait for the node to get up and running
while true
do
  height=$(/bin/celestia-appd query block 1 -n tcp://$CORE_RPC_HOST:$CORE_RPC_PORT 2>/dev/null)
  if [[ -n ${height} ]] ; then
    break
  fi
  echo "Waiting for block 1 to be generated..."
  sleep 5s
done

# check whether to deploy a new contract or use an existing one
if [[ -z "${QGB_CONTRACT}" ]]
then
  export DEPLOY_NEW_CONTRACT=true
  export STARTING_NONCE=earliest
  # expects the script to be mounted to this directory
  /bin/bash /opt/deploy_qgb_contract.sh
fi

# get the address from the `qgb_address.txt` file
QGB_CONTRACT=$(cat /opt/qgb_address.txt)

# init the relayer
/bin/qgb relayer init

# import keys to relayer
/bin/qgb relayer keys evm import ecdsa "${PRIVATE_KEY}" --evm-passphrase 123

# to give time for the bootstrappers to be up
sleep 5s
/bin/qgb relayer start \
  --evm.address="${EVM_ADDRESS}" \
  --core.rpc.host="${CORE_RPC_HOST}" \
  --core.rpc.port="${CORE_RPC_PORT}" \
  --core.grpc.host="${CORE_GRPC_HOST}" \
  --core.grpc.port="${CORE_GRPC_PORT}" \
  --evm.chain-id="${EVM_CHAIN_ID}" \
  --evm.rpc="${EVM_ENDPOINT}" \
  --evm.contract-address="${QGB_CONTRACT}" \
  --p2p.bootstrappers="${P2P_BOOTSTRAPPERS}" \
  --p2p.listen-addr="${P2P_LISTEN}" \
  --evm-passphrase=123
