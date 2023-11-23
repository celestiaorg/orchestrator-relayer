package verifier

import (
	"context"
	"fmt"
	"time"

	"demo/blockchain"
	"demo/network"
)

// Start starts the verifier. Generally, the verifier should be in a separate node/process.
// But for simplicity, we'll just make it in a go routine so that we don't have to handle communication
// between the sequencer and the verifier.
// The verifier's role is whenever it receives a header, it checks if the data was published on Celestia,
// then it compares that data with the block data that it receives.
// If it doesn't receive any block data, or it finds a difference between the two, it can create a fraud proof
// that can be verified in the Blobstream contract verifier.
func Start(ctx context.Context, network *network.Network, headersChan <-chan blockchain.Header, blocksChan <-chan blockchain.Block) {
	fmt.Println("starting verifier")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Second)
			// insert sending transactions logic here
			fmt.Println("doing verifier thing")

			// TODO: Receive header from sequencer
			// TODO: Get the data from Celestia
			// TODO: Receive the block from Sequencer
			// TODO: Verify that the data exists and matches
			// TODO: Create fraud proof otherwise
		}
	}
}
