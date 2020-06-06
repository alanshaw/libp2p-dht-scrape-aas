package lp2p

import (
	"context"
	"sync"

	ds "github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("dht_scrape_aas_lp2p")

// New creates a new libp2p host and DHT for use by a scraper.
func New(ctx context.Context, bootstrapAddrs []string) (host.Host, *dht.IpfsDHT, error) {
	h, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		return nil, nil, err
	}

	log.Infof("created peer with addrs %v", h.Addrs())

	dht := dht.NewDHT(ctx, h, ds.NewMapDatastore())

	bootstrap(ctx, h, bootstrapAddrs)

	return h, dht, nil
}

func bootstrap(ctx context.Context, h host.Host, addrs []string) {
	var wg sync.WaitGroup
	for _, addr := range addrs {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()

			ma, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				log.Error(err)
				return
			}

			ai, err := peer.AddrInfoFromP2pAddr(ma)
			if err != nil {
				log.Error(err)
				return
			}

			if err := h.Connect(ctx, *ai); err != nil {
				log.Error(addr, err)
			}
		}(addr)
	}
	wg.Wait()
}
