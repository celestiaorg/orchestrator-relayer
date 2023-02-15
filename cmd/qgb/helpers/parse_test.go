package helpers

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func TestParseAddrInfos(t *testing.T) {
	testCases := []struct {
		name      string
		addrs     []string
		want      []peer.AddrInfo
		shouldErr bool
	}{
		{
			name:  "empty input",
			addrs: []string{},
			want:  []peer.AddrInfo{},
		},
		{
			name: "valid input",
			addrs: []string{
				"/ip4/127.0.0.1/tcp/8080/p2p/12D3KooWHr2wqFAsMXnPzpFsgxmePgXb8BqpkePebwUgLyZc95bd",
				"/dns4/limani.celestia-devops.dev/tcp/2121/p2p/12D3KooWDgG69kXfmSiHjUErN2ahpUC1SXpSfB2urrqMZ6aWC8NS",
			},
			want: []peer.AddrInfo{
				func() peer.AddrInfo {
					info, _ := peer.AddrInfoFromString("/ip4/127.0.0.1/tcp/8080/p2p/12D3KooWHr2wqFAsMXnPzpFsgxmePgXb8BqpkePebwUgLyZc95bd")
					return *info
				}(),
				func() peer.AddrInfo {
					info, _ := peer.AddrInfoFromString("/dns4/limani.celestia-devops.dev/tcp/2121/p2p/12D3KooWDgG69kXfmSiHjUErN2ahpUC1SXpSfB2urrqMZ6aWC8NS")
					return *info
				}(),
			},
		},
		{
			name: "invalid multiaddr",
			addrs: []string{
				"/ip4/127.0.0.1/tcp/8080",
				"invalid-multiaddr",
			},
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAddrInfos(tmlog.NewNopLogger(), tc.addrs)
			if tc.shouldErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				for i, info := range tc.want {
					assert.Equal(t, info.ID.String(), got[i].ID.String())
					assert.Equal(t, info.Addrs, got[i].Addrs)
				}
			}
		})
	}
}
