package sequencer

import (
	"context"
	"fmt"
	"time"
)

func Start() {
	fmt.Println("starting sequencer")

	ctx := context.Background()

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
			// Also, the keystore can be directly used from the demo/validator/keystore-test.
			// Ganache RPC: ganache:8545
			// Ganache funded account private key: 0x0e9688e585562e828dcbd4f402d5eddf686f947fb6bf75894a85bf008b017401
			// Happy sequencing!
		}
	}
}
