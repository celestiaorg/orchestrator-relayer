# QGB bootstrapper

To bootstrap the QGB P2P network, we use the bootstrapper QGB node type to accept connections from freshly created orchestrators/relayers and share its peer table with them.

## How to run

### Install the QGB binary

Make sure to have the QGB binary installed. Check [the QGB binary page](https://docs.celestia.org/nodes/qgb-binary) for more details.

### Init the store

Before starting the bootstrapper, we will need to init the store:

```ssh
qgb bootstrapper init
```

By default, the store will be created un `~/.bootstrapper`. However, if you want to specify a custom location, you can use the `--home` flag. Or, you can use the following environment variable:

| Variable            | Explanation                         | Default value     | Required |
|---------------------|-------------------------------------|-------------------|----------|
| `BOOTSTRAPPER_HOME` | Home directory for the bootstrapper | `~/.bootstrapper` | Optional |

### Add keys

The P2P private key is optional, and a new one will be generated automatically on the start if none is provided.

The `p2p` sub-command will help you set up this key if you want to use a specific one:

```ssh
qgb bootstrapper p2p  --help
```

### Start the bootstrapper

Now that we have the store initialized, we can start the bootstrapper:

```shell
qgb bootstrapper

QGB P2P network bootstrapper command

Usage:
  qgb bootstrapper [command]

Aliases:
  bootstrapper, bs

Flags:
  -h, --help   help for bootstrapper

Use "qgb bootstrapper [command] --help" for more information about a command.
```

### Open the P2P port

In order for the bootstrapper node to work, you will need to expose the P2P port, which is by default `30000`.

#### Systemd service

An example of a systemd service that can be used for bootstrappers can be found in the orchestrator documentation.
