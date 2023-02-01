package p2p_test

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateHost(t *testing.T) {
	validKeyHex, err := hex.DecodeString("398E0B0478862529D79F21C028317DD181C1E67AF44D099CDC78BD064A13DF671FC02B117A792377BFE4E931045FB47BAB9B236ED3AC43A254A76E9CE9CAD8B8")
	require.NoError(t, err)
	validPrivateKey, err := crypto.UnmarshalEd25519PrivateKey(validKeyHex)
	require.NoError(t, err)

	tests := []struct {
		name            string
		listenMultiAddr string
		privateKey      crypto.PrivKey
		wantErr         bool
	}{
		{
			name:            "valid input",
			listenMultiAddr: "/ip4/127.0.0.1/tcp/0",
			privateKey:      validPrivateKey,
			wantErr:         false,
		},
		{
			name:            "invalid multiaddress",
			listenMultiAddr: "invalid_multiaddress",
			privateKey:      validPrivateKey,
			wantErr:         true,
		},
		{
			name:            "invalid private key",
			listenMultiAddr: "/ip4/127.0.0.1/tcp/0",
			privateKey:      nil,
			wantErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, err := p2p.CreateHost(tt.listenMultiAddr, tt.privateKey)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, host)
				assert.NoError(t, host.Close())
			}
		})
	}
}

func TestConnectHosts(t *testing.T) {
	validKeyHex1, err := hex.DecodeString("398E0B0478862529D79F21C028317DD181C1E67AF44D099CDC78BD064A13DF671FC02B117A792377BFE4E931045FB47BAB9B236ED3AC43A254A76E9CE9CAD8B8")
	require.NoError(t, err)
	validPrivateKey1, err := crypto.UnmarshalEd25519PrivateKey(validKeyHex1)
	require.NoError(t, err)

	validKeyHex2, err := hex.DecodeString("1BFC789DBD7B3CA13B4CF47898088CBB5CE467668DA63740ADF62B06F474452C6E12BD8B0C964D17438B8FEE1AC019D5290E2D4BE5BEED0113E13926581FFCB4")
	require.NoError(t, err)
	validPrivateKey2, err := crypto.UnmarshalEd25519PrivateKey(validKeyHex2)
	require.NoError(t, err)

	host1, err := p2p.CreateHost("/ip4/0.0.0.0/tcp/0", validPrivateKey1)
	require.NoError(t, err)
	require.NotNil(t, host1)

	host2, err := p2p.CreateHost("/ip4/0.0.0.0/tcp/0", validPrivateKey2)
	require.NoError(t, err)
	require.NotNil(t, host2)

	err = host1.Connect(context.Background(), peer.AddrInfo{
		ID:    host2.ID(),
		Addrs: host2.Addrs(),
	})
	require.NoError(t, err)

	haha := host2.Peerstore().PeerInfo(host1.ID()).ID
	assert.Equal(t, haha, host1.ID())

	assert.NoError(t, host1.Close())
	assert.NoError(t, host2.Close())
}
