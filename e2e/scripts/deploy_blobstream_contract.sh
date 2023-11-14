#!/bin/bash

# This script deploys the Blobstream contract and outputs the address to stdout.

# check whether to deploy a new contract or no need
if [[ "${DEPLOY_NEW_CONTRACT}" != "true" ]]
then
  echo "no need to deploy a new Blobstream contract. exiting..."
  exit 0
fi

# check if environment variables are set
if [[ -z "${EVM_CHAIN_ID}" || -z "${PRIVATE_KEY}" ]] || \
   [[ -z "${CORE_RPC_HOST}" || -z "${CORE_RPC_PORT}" ]] || \
   [[ -z "${CORE_GRPC_HOST}" || -z "${CORE_GRPC_PORT}" ]] || \
   [[ -z "${EVM_ENDPOINT}" || -z "${STARTING_NONCE}" ]]
then
  echo "Environment not setup correctly. Please set:"
  echo "EVM_CHAIN_ID, PRIVATE_KEY, CORE_RPC_HOST, CORE_RPC_PORT, CORE_GRPC_HOST, CORE_GRPC_PORT, EVM_ENDPOINT, STARTING_NONCE variables"
  exit 1
fi

# wait for the node to get up and running
while true
do
  # verify that the node is listening on gRPC
  nc -z -w5 "$CORE_GRPC_HOST" "$CORE_GRPC_PORT"
  result=$?
  if [ "${result}" != "0" ]; then
    echo "Waiting for node gRPC to be available ..."
    sleep 1s
    continue
  fi

  height=$(/bin/celestia-appd query block 1 -n tcp://${CORE_RPC_HOST}:${CORE_RPC_PORT} 2>/dev/null)
  if [[ -n ${height} ]] ; then
    break
  fi
  echo "Waiting for block 1 to be generated..."
  sleep 1s
done

# wait for the evm node to start
while true
do
    status_code=$(curl --write-out '%{http_code}' --silent --output /dev/null \
                      --location --request POST ${EVM_ENDPOINT} \
                      --header 'Content-Type: application/json' \
                      --data-raw "{
                  	    \"jsonrpc\":\"2.0\",
                  	    \"method\":\"eth_blockNumber\",
                  	    \"params\":[],
                  	    \"id\":${EVM_CHAIN_ID}}")
    if [[ "${status_code}" -eq 200 ]] ; then
      break
    fi
    echo "Waiting for ethereum node to be up..."
    sleep 1s
done

# import keys to deployer
/bin/blobstream deploy keys evm import ecdsa "${PRIVATE_KEY}" --evm.passphrase=123

echo "deploying Blobstream contract..."

/bin/blobstream deploy \
  --evm.chain-id "${EVM_CHAIN_ID}" \
  --evm.account "${EVM_ACCOUNT}" \
  --core.rpc="${CORE_RPC_HOST}:${CORE_RPC_PORT}" \
  --core.grpc="${CORE_GRPC_HOST}:${CORE_GRPC_PORT}" \
  --grpc.insecure \
  --starting-nonce "${STARTING_NONCE}" \
  --evm.rpc "${EVM_ENDPOINT}" \
  --evm.passphrase=123 2> /opt/output

echo $(cat /opt/output)

cat /opt/output | grep "deployed" | awk '{ print $6 }' | grep -o '0x.*' > /opt/blobstream_address.txt
