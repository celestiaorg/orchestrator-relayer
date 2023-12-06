package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"demo/blockchain"
	"demo/network"
	"demo/sequencer"
	"demo/verifier"
)

func main() {
	ctx := context.Background()
	headersChan := make(chan blockchain.Header)
	blocksChan := make(chan blockchain.Block, 100)
	net, err := network.NewNetwork(ctx, "http://ganache:8545", "tcp://validator:26657", "validator:9090")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	w := sync.WaitGroup{}
	w.Add(1)
	go sequencer.Start(ctx, net, headersChan, blocksChan)

	w.Add(2)
	go verifier.Start(ctx, net, headersChan, blocksChan)

	w.Wait()
}
