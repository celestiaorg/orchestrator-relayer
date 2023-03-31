#!/bin/bash

# This script waits  for the validator to be created before starting the orchestrator

# check if environment variables are set
if [[ -z "${MONIKER}" || -z "${PRIVATE_KEY}" ]] || \
   [[ -z "${TENDERMINT_RPC}" || -z "${CELESTIA_GRPC}" ]] || \
   [[ -z "${P2P_IDENTITY}" || -z "${P2P_LISTEN}" ]]
then
  echo "Environment not setup correctly. Please set:"
  echo "MONIKER, PRIVATE_KEY, TENDERMINT_RPC, CELESTIA_GRPC, P2P_IDENTITY, P2P_LISTEN variables"
  exit 1
fi

# install needed dependencies
apk add curl

# wait for the validator to be created before starting the orchestrator
VAL_ADDRESS=$(celestia-appd keys show ${MONIKER} --keyring-backend test --bech=val --home /opt -a)
while true
do
  # verify that the node is listening on gRPC
  nc -z -w5 $(echo $CELESTIA_GRPC | cut -d : -f 1) $(echo $CELESTIA_GRPC | cut -d : -f 2)
  result=$?
  if [ "${result}" != "0" ]; then
    echo "Waiting for node gRPC to be available ..."
    sleep 5s
    continue
  fi

  # verify if RPC is running and the validator was created
  output=$(celestia-appd query staking validator ${VAL_ADDRESS} --node $TENDERMINT_RPC 2>/dev/null)
  if [[ -n "${output}" ]] ; then
    break
  fi
  echo "Waiting for validator to be created..."
  sleep 5s
done

# initialize orchestrator
/bin/qgb orch init

# start orchestrator
if [[ -z "${P2P_BOOTSTRAPPERS}" ]]
then
  /bin/qgb orchestrator start \
    -p=/opt \
    -x=qgb-e2e \
    -d="${PRIVATE_KEY}" \
    -t="${TENDERMINT_RPC}" \
    -c="${CELESTIA_GRPC}" \
    -p="${P2P_IDENTITY}" \
    -q="${P2P_LISTEN}"
else
  # to give time for the bootstrappers to be up
  sleep 5s
  /bin/qgb orchestrator start \
    -p=/opt \
    -x=qgb-e2e \
    -d="${PRIVATE_KEY}" \
    -t="${TENDERMINT_RPC}" \
    -c="${CELESTIA_GRPC}" \
    -b="${P2P_BOOTSTRAPPERS}" \
    -p="${P2P_IDENTITY}" \
    -q="${P2P_LISTEN}"
fi
