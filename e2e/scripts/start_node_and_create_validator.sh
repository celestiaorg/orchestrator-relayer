#!/bin/bash

# This script starts a Celestia-app, creates a validator with the provided parameters, then
# keeps running it validating blocks.

# check if environment variables are set
if [[ -z "${CELESTIA_HOME}" || -z "${MONIKER}" || -z "${EVM_ACCOUNT}" || -z "${AMOUNT}" ]]
then
  echo "Environment not setup correctly. Please set: CELESTIA_HOME, MONIKER, EVM_ACCOUNT, AMOUNT variables"
  exit 1
fi

# create necessary structure if doesn't exist
if [[ ! -f ${CELESTIA_HOME}/data/priv_validator_state.json ]]
then
    mkdir "${CELESTIA_HOME}"/data
    cat <<EOF > ${CELESTIA_HOME}/data/priv_validator_state.json
{
  "height": "0",
  "round": 0,
  "step": 0
}
EOF
fi

{
  # wait for the node to get up and running
  while true
  do
    status_code=$(curl --write-out '%{http_code}' --silent --output /dev/null localhost:26657/status)
    if [[ "${status_code}" -eq 200 ]] ; then
      break
    fi
    echo "Waiting for node to be up..."
    sleep 2s
  done

  VAL_ADDRESS=$(celestia-appd keys show "${MONIKER}" --keyring-backend test --bech=val --home /opt -a)
  # keep retrying to create a validator
  while true
  do
    # create validator
    celestia-appd tx staking create-validator \
      --amount="${AMOUNT}" \
      --pubkey="$(celestia-appd tendermint show-validator --home "${CELESTIA_HOME}")" \
      --moniker="${MONIKER}" \
      --chain-id="qgb-e2e" \
      --commission-rate=0.1 \
      --commission-max-rate=0.2 \
      --commission-max-change-rate=0.01 \
      --min-self-delegation=1000000 \
      --from="${MONIKER}" \
      --keyring-backend=test \
      --home="${CELESTIA_HOME}" \
      --broadcast-mode=block \
      --fees="300000utia" \
      --yes
    output=$(celestia-appd query staking validator "${VAL_ADDRESS}" 2>/dev/null)
    if [[ -n "${output}" ]] ; then
      break
    fi
    echo "trying to create validator..."
    sleep 1s
  done

  # Register the validator EVM address
  celestia-appd tx qgb register \
    "${VAL_ADDRESS}" \
    "${EVM_ACCOUNT}" \
    --from "${MONIKER}" \
    --home "${CELESTIA_HOME}" \
    --fees "30000utia" -b block \
    --chain-id="qgb-e2e" \
    --yes
} &

# start node
celestia-appd start \
--home="${CELESTIA_HOME}" \
--moniker="${MONIKER}" \
--p2p.persistent_peers=40664e2a69d5557b0d35ec8b9d423785b12579b7@core0:26656 \
--rpc.laddr=tcp://0.0.0.0:26657
