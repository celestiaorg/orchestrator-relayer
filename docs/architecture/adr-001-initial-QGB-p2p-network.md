# ADR 001: Initial QGB P2P network

## Context

The QGB V2 will rely on a P2P network to exchange attestations signatures. This comes after deciding that using the state will result in state bloat, and using block data will always have an opportunity cost for validators. Thus, we will rely on a P2P network for exchanging attestations' signatures between orchestrators and relayers.

An attestation signature can be identified by the `(nonce,orchestrator,type)` tuple:

- `nonce`: nonce of the attestation
- `orchestrator`: orchestrator address
- `type`: data commitment confirm or valset confirm

The issue with using a P2P network is that there is no way of knowing the CID of a signature beforehand. So, relayers will not be able to get the confirms directly from the network.

## Possible solutions

The standard solution is to use Libp2p kadDHT implementation for discovery and routing, then using a Bitswap exchange to publish attestation signatures and get them using CIDs. However, in our case, the CIDs will be unknown until the moment of signature creation making it impossible to rely on Bitswap alone. So, we will need to use another mechanism to know the CIDs or to get them.

### Option 1: DHT + Bitswap

Use the DHT as an indexing engine for the confirms and then getting the confirms using bitswap:

```go
struct SignatureIndex {
  CID cid.CID
  IndexSignature []byte // optional: we can write a validator to validate this signature: <https://github.com/libp2p/go-libp2p-record/blob/f093f9649af5edc2edcb3c262bd2d2a4b022d601/validator.go#L27>
}
```

**Producer**: Orchestrator, which will be signing attestations:

- Sign an attestation
- Create the attestation confirm struct
- Post it to the blockstore
- Create a `SignatureIndex`
- Put the Signature index in the DHT.

```go
attestationConfirm := AttestationConfirm {
...
}
cid, err := cidlink.DefaultLinkSystem().Store(..., attestationConfirm)
signatureIndex := SignatureIndex {
  CID: cid,
  // other fields
}
indexKey := "<nonce,orchestrator_addr,type>"
dht.PutValue(ctx, indexKey, signatureIndex)
```

The DHT validator should run `ValidateBasics` on the `SignatureIndex` because if a value with a certain key was posted, it can't be overwritten.

**Consumers**: Relayer, which will be querying attestations:

- Create the `SignatureIndex`. All the fields would be known when querying for a certain signature.
- Query the signature index from the DHT
- Query the confirm from bitswap

```go
indexKey := "<nonce,orchestrator_addr,type>"
index, err := dht.GetValue(ctx, indexKey)
confirm, err := exchange.GetBlock(ctx, index.CID)
```

### Limitations

Using the above approach will give us the following limitations:

- Consumers will have to know when to query the network:
  - This won't be an issue since we can define a delay to start querying: the time when the attestation request was created in the state machine + a few minutes
- We will need to handle republishing the signature indexes periodically. This is because the DHT has a delay of 48 hours before it deletes the values put on it.

### Options 2: DHT alone

Instead of using bitswap along with the DHT, we could use only the DHT to keep the signatures as values.

This will have the same limitations as option 1. However, these shouldn't be an issue with our use case.

Also, it won't be much of a difference between using the DHT as an indexing mechanism, in terms of storage as the structs are almost similar.

### Option 3: [Conflict-Free Replicated Data Types](https://github.com/ipfs/go-ds-crdt)

- Merkle CRDT uses the following components:
  - A DAG-syncer: which syncs DAG nodes between nodes and publishes new ones. IPFS will be used for this.
  - A broadcaster: which sends messages to all the peers. This will be LibP2P PubSub.
- Limitations:
  - Ever-growing DAG: shouldn't be a problem in our case since we only have small unrelated key values.
  - Merkle clock sorting: having access to a logical clock to sort events as they happen. This shouldn't cause us any issue since the signatures are not order dependent.
  - TBD.
- We should take a deeper look at the repo [go-ds-crdt](https://github.com/ipfs/go-ds-crdt) as there isn't much activity:
  - low number of issues.
  - has a single contributor doing most of the work.
  - No production-grade project is using it.
  - depends on [ipfs-lite](https://github.com/hsanjuan/ipfs-lite) which is maintained by the same person. The alternative would be to use the official IPFS daemon.

### Option 4: IPNS

Instead of using the DHT for storing the indexes, we could use IPNS where each peer would publish its list of signatures, and upon every signature, update what it's publishing.

However, this approach has the following downsides:

- Peers must maintain the same `peerID`, the same identity. We could enforce that the identity of the peer is the same as their orchestrator private key.
- If a peer is offline, the network won't have access to the signatures until it is up again.

## Decision

Implement **option 2**, i.e. relying on the DHT solely to exchange attestations for now, and then change in the future if there is a reason.

## Detailed design

### Data structures

The following structs will be stored as DHT values:

```go
type DataCommitmentConfirm struct {
  Signature string
  EthAddress string
  Commitment string
}
```

and:

```go
type ValsetConfirm struct {
  EthAddress string
  Signature string
}
```

These will be submitted by orchestrators who sign attestations, and will be queried by relayers who submit the confirms to the target EVM chain.

### Validators

To add an extra layer of security when exchanging the data, we will define validators which will validate the confirms before storing/querying them:

```go
type ValsetValidator struct {
}

func (vv ValsetValidator) Validate(key string, value []byte) error {
  // TODO Should verify that the valset is valid, i.e. running stateless checks on it.
  // The checks should include:
  // - Correct signature verification
  // - Correct fields checks. Example, checking if an address field as a correctly formatted address.
  return nil
}

func (vv ValsetValidator) Select(key string, values [][]byte) (int, error) {
  // TODO Should run the same stateless checks as the `Validate` function to avoid querying
  // faulty values.
  return 0, nil
}
```

A similar one should be created for `DataCommitmentConfirm`s.

### Registering validators

To register a validator, we should do the following when creating the DHT:

```go
dht.New(
    ...
    dht.NamespacedValidator("vs", ValsetValidator{}),
    dht.NamespacedValidator("dc", DataCommitmentValidator{}),
  )
```

This way, we can add values to the DHT following their types:

- `"/vs/<key>"`: refers to a valset confirm with key `<key>`.
- `"/dc/<key>"`: refers to a data commitment confirm with key `<key>`.

### Keys

To have predictable confirms values keys, we can define the keys to be: `"<nonce,orchestrator_addr>"`:

- `nonce`: is the universal nonce of the attestations being signed
- `orchestrator_addr`: is the `celes1` address of the orchestrator that signed the attestation.

## Status

Proposed.

## Consequences

### Positive

- Use a well tested/maintained project, libP2P implementation of DHT.

### Neutral

- We might have to re-implement some functionality like: republishing values every 48 hours as they disappear automatically after that period.
