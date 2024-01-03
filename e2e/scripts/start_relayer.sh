#!/bin/bash

# This script runs the Blobstream relayer with the ability to deploy a new Blobstream contract or
# pass one as an environment variable BLOBSTREAM_CONTRACT

# check if environment variables are set
if [[ -z "${EVM_CHAIN_ID}" || -z "${PRIVATE_KEY}" ]] || \
   [[ -z "${CORE_GRPC_HOST}" || -z "${CORE_GRPC_PORT}" ]] || \
   [[ -z "${CORE_RPC_HOST}" || -z "${CORE_RPC_PORT}" ]] || \
   [[ -z "${EVM_ENDPOINT}" || -z "${P2P_BOOTSTRAPPERS}" ]] || \
   [[ -z "${P2P_LISTEN}" || -z "${METRICS_ENDPOINT}" ]]
then
  echo "Environment not setup correctly. Please set:"
  echo "EVM_CHAIN_ID, PRIVATE_KEY, CORE_GRPC_HOST, CORE_GRPC_PORT, CORE_RPC_HOST, CORE_RPC_PORT, EVM_ENDPOINT, P2P_BOOTSTRAPPERS, P2P_LISTEN, METRICS_ENDPOINT variables"
  exit 1
fi

echo "starting relayer..."

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

# this will introduce flakiness but it's gonna be complicated to wait for validators to be created
# and also waiting for them to change their addresses in bash. Also, depending on the testing scenarios,
# the network topology varies. So, the best we can do now is sleep.
sleep 120s

# check whether to deploy a new contract or use an existing one
if [[ -z "${BLOBSTREAM_CONTRACT}" ]]
then
  export DEPLOY_NEW_CONTRACT=true
  export STARTING_NONCE=latest
  # expects the script to be mounted to this directory
  /bin/bash /opt/deploy_blobstream_contract.sh
fi

# get the address from the `blobstream_address.txt` file
BLOBSTREAM_CONTRACT=$(cat /opt/blobstream_address.txt)

# init the relayer
/bin/blobstream relayer init

# import keys to relayer
/bin/blobstream relayer keys evm import ecdsa "${PRIVATE_KEY}" --evm.passphrase 123

# to give time for the bootstrappers to be up
sleep 5s
/bin/blobstream relayer start \
  --evm.account="${EVM_ACCOUNT}" \
  --core.rpc="${CORE_RPC_HOST}:${CORE_RPC_PORT}" \
  --core.grpc="${CORE_GRPC_HOST}:${CORE_GRPC_PORT}" \
  --grpc.insecure \
  --evm.chain-id="${EVM_CHAIN_ID}" \
  --evm.rpc="${EVM_ENDPOINT}" \
  --evm.contract-address="${BLOBSTREAM_CONTRACT}" \
  --p2p.bootstrappers="${P2P_BOOTSTRAPPERS}" \
  --p2p.listen-addr="${P2P_LISTEN}" \
  --evm.passphrase=123 \
  --metrics \
  --metrics.endpoint="${METRICS_ENDPOINT}" \
  --log.level=debug
