#!/bin/bash

# This script starts core0

# install needed dependencies
apk add jq coreutils

# set the genesis time to current time for pruning to work properly
new_time=$(date -u +"%Y-%m-%dT%H:%M:%S.%N")"Z"
jq --arg new_time "$new_time" '.genesis_time = $new_time' /opt/config/genesis_template.json > /opt/config/genesis.json

if [[ ! -f /opt/data/priv_validator_state.json ]]
then
    mkdir /opt/data
    cat <<EOF > /opt/data/priv_validator_state.json
{
  "height": "0",
  "round": 0,
  "step": 0
}
EOF
fi

/bin/celestia-appd start \
  --moniker core0 \
  --rpc.laddr tcp://0.0.0.0:26657 \
  --home /opt
