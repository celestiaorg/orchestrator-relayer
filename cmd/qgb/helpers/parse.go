package helpers

import (
	"github.com/libp2p/go-libp2p/core/peer"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

// ParseAddrInfos converts strings to AddrInfos
func ParseAddrInfos(logger tmlog.Logger, addrs []string) ([]peer.AddrInfo, error) {
	infos := make([]peer.AddrInfo, 0, len(addrs))
	for _, addr := range addrs {
		info, err := peer.AddrInfoFromString(addr)
		if err != nil {
			logger.Error("parsing info from multiaddr", "addr", addr, "err", err)
			return nil, err
		}
		infos = append(infos, *info)
	}
	return infos, nil
}
