# Mock QGB

This repo contains a unified orchestrator-relayer that uses a mock valset to post commitments from a celestia network.
The cli command can be used to deploy the contract at any particular nonce (or "earliest", "latest") and can be used to
spin up the orchestrator-relayer to post commitments to any EVM chain

## Install orchrelayer
```
make install
```

## Deploy contract
For example, to deploy the contract for the latest nonce from mocha to goerli
```
orchrelayer deploy --celes-chain-id test --evm-priv-key `priv-key` --evm-chain-id 5 --starting-nonce latest --evm-rpc https://rpc.ankr.com/eth_goerli
```

Post commitments
For example, to post commitments to the contract deployed above from mocha to goerli
```
orchrelayer orchestrator-relayer --evm-chain-id 5 --evm-rpc https://rpc.ankr.com/eth_goerli --contract-address 0x29024e5D1a6C460E82F2e045c3AEf9e46A1d8595 --evm-priv-key `priv-key` --celes-chain-id mocha
```