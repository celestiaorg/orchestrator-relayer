# QGB Orchestrator

<!-- markdownlint-disable MD013 -->

The role of the orchestrator is to sign attestations using its corresponding validator EVM private key. These attestations are generated within the QGB module of the Celestia-app state machine. To learn more about what attestations are, you can refer to this [link](https://github.com/celestiaorg/celestia-app/tree/main/x/qgb).

## How it works

The orchestrator does the following:

1. Connect to a Celestia-app full node or validator node via RPC and gRPC and wait for new attestations
2. Once an attestation is created inside the QGB state machine, the orchestrator queries it.
3. After getting the attestation, the orchestrator signs it using the provided EVM private key. The private key should correspond to the EVM address provided when creating the validator. More details in [here](https://docs.celestia.org/nodes/validator-node/#setup-qgb-keys).
4. Then, the orchestrator pushes its signature to the P2P network it is connected to, via adding it as a DHT value.
5. Listen for new attestations and go back to step 2.

## How to run

### Install the QGB binary

Make sure to have the QGB binary installed. Check [here](https://github.com/celestiaorg/orchestrator-relayer/blob/main/README.md) for more details.

### Init the store

Before starting the orchestrator, we will need to init the store:

```ssh
qgb orchestrator init
```

By default, the store will be created under `~/.orchestrator`. However, if you want to specify a custom location, you can use the `--home` flag. Or, you can use the following environment variable:

| Variable            | Explanation                           | Default value     | Required |
|---------------------|---------------------------------------|-------------------|----------|
| `ORCHESTRATOR_HOME` | Home directory for the orchestrator   | `~/.orchestrator` | Optional |

### Add keys

In order for the orchestrator to start, it will need two private keys:

- EVM private key
- P2P private key

The EVM private key is the most important one since it needs to correspond to the EVM address provided when creating the validator.

The P2P private key is optional, and a new one will be generated automatically on the start if none is provided.

The `keys` command will help you set up these keys:

```ssh
qgb orchestrator keys  --help
```

To add an EVM private key, check the next section.

#### EVM key

Because EVM keys are important, we provide a keystore that will help manage them. The keystore uses a file system keystore protected by a passphrase to store and open private keys.

To import your EVM private key, there is the `import` subcommand to assist you with that:

```ssh
qgb orchestrator keys evm import --help
```

This subcommand allows you to either import a raw ECDSA private key provided as plaintext, or import it from a file. The files are JSON keystore files encrypted using a passphrase like in [here](https://geth.ethereum.org/docs/developers/dapp-developer/native-accounts).

After adding the key, you can check that it's added via running:

```ssh
qgb orchestrator keys evm list
```

For more information about the `keys` command, check the `keys` documentation in [here](https://github.com/celestiaorg/orchestrator-relayer/blob/main/docs/keys.md).

### Start the orchestrator

Now that we have the store initialized, we can start the orchestrator. Make sure you have your Celestia-app node RPC and gRPC accessible, and able to connect to the P2P network bootstrappers.

The orchestrator accepts the following flags:

```ssh
qgb orchestrator start --help

Starts the QGB orchestrator to sign attestations

Usage:
  qgb orchestrator start <flags> [flags]

Flags:
  -c, --celes-grpc string          Specify the grpc address (default "localhost:9090")
  -t, --celes-http-rpc string      Specify the rest rpc address (default "http://localhost:26657")
  -d, --evm-address string         Specify the EVM account address to use for signing (Note: the private key should be in the keystore)
  -h, --help                       help for start
      --home string                The qgb orchestrator home directory
  -b, --p2p-bootstrappers string   Comma-separated multiaddresses of p2p peers to connect to
  -q, --p2p-listen-addr string     MultiAddr for the p2p peer to listen on (default "/ip4/0.0.0.0/tcp/30000")
  -p, --p2p-nickname string        Nickname of the p2p private key to use (if not provided, an existing one from the p2p store or a newly generated one will be used)
      --evm-passphrase string          the account passphrase (if not specified as a flag, it will be asked interactively)
```

To start the orchestrator in the default home directory, run the following:

```ssh
qgb orchestrator start \
    -c localhost:9090 \
    -t http://localhost:26657 \
    -d 0x966e6f22781EF6a6A82BBB4DB3df8E225DfD9488 \
    -b /ip4/127.0.0.1/tcp/30001/p2p/12D3KooWFFHahpcZcuqnUhpBoX5fJ68Qm5Hc8dxiBcX1oo46fLxh \
    -q /ip4/0.0.0.0/tcp/30000
```

And, you will be prompted to enter your EVM key passphrase so that the orchestrator can use it to sign attestations. Make sure that it's the EVM address that was provided when creating the validator. If not, then the orchestrator will not sign, and you will keep seeing a "validator not part of valset" warning message. If you see such message, first verify that your validator is part of the active validator set. If so, then probably the EVM address provided to the orchestrator is not the right one, and you should check which EVM address is registered to your validator. Check the next section for more information.

If you no longer have access to your EVM address, you could always edit your validator with a new EVM address. This can be done through the `edit-validator` command. Check the next section.

#### Edit validator

If your validator was created using an EVM address that you don't have access to, you can always edit it using the `edit-validator` command.

First, you should get your validator `valoper` address. To do so, run the following:

```ssh
celestia-appd keys show <validator_account> --bech val
```

This assumes that you're using the default home directory, the default keystore etc. If not, make sure to add the flags that correspond to your situation.

Then, you should get your validator to verify which EVM address is attached to it:

```ssh
celestia-appd query staking validator <validator_valoper_address>
```

And check the `evm_address` field if it has an address that you want to use to sign attestations. If not, let's proceed to change it.

Note: Please double-check the parameters of the following command before running it, as it may have persistent effects.

```ssh
celestia-appd tx staking edit-validator --evm-address=<new_evm_address> --from=<validator_account> --fees 210utia
```

Example command output:

```ssh
auth_info:
  fee:
    amount:
    - amount: "210"
      denom: utia
    gas_limit: "210000"
    granter: ""
    payer: ""
  signer_infos: []
  tip: null
body:
  extension_options: []
  memo: ""
  messages:
  - '@type': /cosmos.staking.v1beta1.MsgEditValidator
    commission_rate: null
    description:
      details: '[do-not-modify]'
      identity: '[do-not-modify]'
      moniker: '[do-not-modify]'
      security_contact: '[do-not-modify]'
      website: '[do-not-modify]'
    evm_address: 0x27a1F8CE94187E4b043f4D57548EF2348Ed556c7
    min_self_delegation: null
    validator_address: celestiavaloper1vr6j8mq6aaxr5mw9sld3a75afjr4rytp42zy6h
  non_critical_extension_options: []
  timeout_height: "0"
signatures: []
confirm transaction before signing and broadcasting [y/N]: y
code: 0
codespace: ""
data: ""
events: []
gas_used: "0"
gas_wanted: "0"
height: "0"
info: ""
logs: []
raw_log: '[]'
timestamp: ""
tx: null
txhash: 25864170DDE40F51C0C38BCF5B22BBC015637F56AA1E2DFDA6CE51F2D5860579
```

Now, you can verify that the EVM address has been updated using the following command:

```ssh
celestia-appd query staking validator <validator_valoper_address>
```

Now, you can restart the orchestrator, and it should start signing.
