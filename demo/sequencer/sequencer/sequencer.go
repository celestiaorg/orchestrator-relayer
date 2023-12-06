package sequencer

import (
	"context"
	"fmt"
	"time"

	"demo/blockchain"
	"demo/network"
)

func Start(ctx context.Context, network *network.Network, headersChan chan<- blockchain.Header, blocksChan chan<- blockchain.Block) {
	fmt.Println("starting sequencer")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Second)
			// insert sending transactions logic here
			fmt.Println("doing something")

			// In this setup the following endpoints can be helpful:
			// RPC: tcp://validator:26657
			// gRPC: validator:9090
			// Celestia funded account:
			/*
				-----BEGIN TENDERMINT PRIVATE KEY-----
				kdf: bcrypt
				salt: 643A75077C2ED9FE77FD9423D9E9F311
				type: secp256k1

				XwNYAUs53nIeo43oZTzoWny2zw7RImFpBLsPFUw7xz6A9juwyrNzvSByVevbYLhD
				vLmXAojArP/gftMzhjbKOtEBN8GeMyfNycrGFp0=
				=9zVO
				-----END TENDERMINT PRIVATE KEY-----
			*/
			// Passphrase: blobstream-demo
			// Also, the keystore can be directly added as a volume to the sequencer container
			// and used from the keystore-test directly.
			// Ganache RPC: ganache:8545
			// Ganache funded account private key: 0x0e9688e585562e828dcbd4f402d5eddf686f947fb6bf75894a85bf008b017401
			// Happy sequencing!

			// TODO: Create transactions
			// TODO: Create Block
			// TODO: Submit block to Celestia
			// TODO: Submit header to settlement contract:
			// The settlement contract needs to be implemented and deployed to Ganache. It will
			// inherit from the DAVerifier: https://github.com/celestiaorg/blobstream-contracts/blob/master/src/lib/verifier/DAVerifier.sol
			// to be able to process inclusion proofs.
			// TODO: Share the header with the verifier
			headersChan <- blockchain.Header{}
			blocksChan <- blockchain.Block{}
		}
	}
}
