# orchestrator-relayer

Contains the implementation of the Blobstream orchestrator and relayer.

The orchestrator is the software that signs Blobstream attestations, and the relayer is the one that relays them to the target EVM chain.

For a high-level overview of how Blobstream works, check [here](https://github.com/celestiaorg/quantum-gravity-bridge/tree/76efeca0be1a17d32ef633c0fdbd3c8f5e4cc53f#how-it-works) and [here](https://blog.celestia.org/celestiums/).

## Install

1. [Install Go](https://go.dev/doc/install) 1.21 
2. Clone this repo
3. Install the Blobstream CLI

 ```shell
make install
```

## Usage

```sh
# Print help
blobstream --help
```

## How to run

If you are a Celestia-app validator, all you need to do is run the orchestrator. Check [here](https://github.com/celestiaorg/orchestrator-relayer/blob/main/docs/orchestrator.md) for more details.

If you want to post commitments on an EVM chain, you will need to deploy a new Blobstream contract and run a relayer, or run a relayer to an already deployed Blobstream contract. Check [relayer docs](https://github.com/celestiaorg/orchestrator-relayer/blob/main/docs/relayer.md) and [deployment docs](https://github.com/celestiaorg/orchestrator-relayer/blob/main/docs/deploy.md) for more information.

Note: the Blobstream P2P network is a separate network from the consensus or the data availability one. Thus, you will need its specific bootstrappers to be able to connect to it.

## Contributing

### Tools

1. Install [golangci-lint](https://golangci-lint.run/usage/install/)
2. Install [markdownlint](https://github.com/DavidAnson/markdownlint)

### Helpful Commands

```sh
# Build a new orchestrator-relayer binary and output to build/blobstream
make build

# Run tests
make test

# Format code with linters (this assumes golangci-lint and markdownlint are installed)
make fmt
```

## Useful links

The smart contract implementation is in [blobstream-contracts](https://github.com/celestiaorg/blobstream-contracts).

The state machine implementation is in [x/blobstream](https://github.com/celestiaorg/celestia-app/tree/main/x/blobstream).

Blobstream ADRs are in the [docs](https://github.com/celestiaorg/celestia-app/tree/main/docs/architecture).

Blobstream design explained in this [blog](https://blog.celestia.org/celestiums).
