package p2p

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
)

// CreateHost Creates a LibP2P host using a listen address and a private key.
// The listen address is a MultiAddress of the format: /ip4/0.0.0.0/tcp/0
// Using port 0 means that it will use a random open port.
// The private key shouldn't be nil.
func CreateHost(listenMultiAddr string, privateKey crypto.PrivKey) (host.Host, error) {
	multiAddr, err := multiaddr.NewMultiaddr(listenMultiAddr)
	if err != nil {
		return nil, err
	}

	if privateKey == nil {
		return nil, ErrNilPrivateKey
	}

	h, err := libp2p.New(
		libp2p.ListenAddrs(multiAddr),
		libp2p.Identity(privateKey),
		libp2p.EnableNATService(),
		// TODO investigate if more options are needed
	)
	if err != nil {
		return nil, err
	}

	return h, nil
}
