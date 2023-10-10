---
sidebar_label: Deploy the BlobStream contract
description: Learn how to deploy the BlobStream smart contract.
---

# Deploy the BlobStream contract

<!-- markdownlint-disable MD013 -->

The `deploy` is a helper command that allows deploying the BlobStream smart contract to a new EVM chain:

```ssh
bstream deploy --help

Deploys the BlobStream contract and initializes it using the provided Celestia chain

Usage:
  bstream deploy <flags> [flags]
  bstream deploy [command]

Available Commands:
  keys        BlobStream keys manager
```

## How to run

### Install the BlobStream binary

Make sure to have the BlobStream binary installed. Check [the BlobStream binary page](https://docs.celestia.org/nodes/blobstream-binary) for more details.

### Add keys

In order to deploy a BlobStream smart contract, you will need a funded EVM address and its private key. The `keys` command will help you set up this key:

```ssh
bstream deploy keys  --help
```

To import your EVM private key, there is the `import` subcommand to assist you with that:

```ssh
bstream deploy keys evm import --help
```

This subcommand allows you to either import a raw ECDSA private key provided as plaintext, or import it from a file. The files are JSON keystore files encrypted using a passphrase like in [this example](https://geth.ethereum.org/docs/developers/dapp-developer/native-accounts).

After adding the key, you can check that it's added via running:

```ssh
bstream deploy keys evm list
```

For more information about the `keys` command, check [the `keys` documentation](https://docs.celestia.org/nodes/blobstream-keys).

### Deploy the contract

Now, we can deploy the BlobStream contract to a new EVM chain:

```ssh
blobstream deploy \
  --evm.chain-id 4 \
  --evm.contract-address 0x27a1F8CE94187E4b043f4D57548EF2348Ed556c7 \
  --core.grpc.host localhost \
  --core.grpc.port 9090 \
  --starting-nonce latest \
  --evm.rpc http://localhost:8545
```

The `latest` can be replaced by the following:

- `latest`: to deploy the BlobStream contract starting from the latest validator set.
- `earliest`: to deploy the BlobStream contract starting from genesis.
- `nonce`: you can provide a custom nonce on where you want the BlobStream to start. If the provided nonce is not a `Valset` attestation, then the one before it will be used to deploy the BlobStream smart contract.

And, now you will see the BlobStream smart contract address in the logs along with the transaction hash.
