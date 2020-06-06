package scraper

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
)

var log = logging.Logger("dht_scrape_aas")

const (
	// totalRounds is the number of rounds performed by each host.
	totalRounds = 15
	// roundInterval is the sleep time between rounds.
	roundInterval = time.Second * 10
	// totalKeys is the number of random keys to get closest peers for, per round.
	totalKeys = 15
	// keySearchTimeout is the maximum time a closest peers query can run for.
	keySearchTimeout = time.Second * 30
)

// PeerStat contains information about a seen peer.
type PeerStat struct {
	PeerID       string   `json:"peerID"`
	Address      string   `json:"address"`
	Protocols    []string `json:"protocols"`
	AgentVersion string   `json:"agentVersion"`
}

// Scraper is a DHT scraper.
type Scraper interface {
	// Scrape starts a new scraping process.
	Scrape(ctx context.Context) <-chan PeerStat
}

type scraper struct{}

// New creates a new DHT scraper.
func New() (Scraper, error) {
	return &scraper{}, nil
}

// Scrape starts a new scraping process.
func (n *scraper) Scrape(ctx context.Context) <-chan PeerStat {
	ch := make(chan PeerStat)

	go func() {
		for {
			if err := runScrape(ctx, ch); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Error("scrape failed: ", err)
			}
		}
	}()

	return ch
}

func runScrape(ctx context.Context, ch chan PeerStat) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	h, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		return err
	}

	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			// wait for the info to get into the peerstore...
			t := time.NewTimer(time.Second)
			select {
			case <-t.C:
			case <-ctx.Done():
				return
			}

			pstat := PeerStat{
				PeerID:  c.RemotePeer().String(),
				Address: c.RemoteMultiaddr().String(),
			}

			av, err := n.Peerstore().Get(c.RemotePeer(), "AgentVersion")
			if err == nil {
				pstat.AgentVersion = fmt.Sprint(av)
			}

			protos, _ := n.Peerstore().GetProtocols(c.RemotePeer())
			pstat.Protocols = protos

			select {
			case ch <- pstat:
			default:
				log.Warn("dropped peer stat due to full channel", pstat)
			}
		},
	})

	log.Infof("created peer with addrs %v", h.Addrs())

	dht := dht.NewDHT(ctx, h, ds.NewMapDatastore())

	bootstrap(ctx, h)

	for i := 0; i < totalRounds; i++ {
		log.Infof("starting scrape round %d/%d\n", i+1, totalRounds)
		if err := runScrapeRound(ctx, h, dht); err != nil {
			return err
		}
		time.Sleep(roundInterval)
	}
	return nil
}

func bootstrap(ctx context.Context, h host.Host) {
	var wg sync.WaitGroup
	for _, addr := range DefaultBootstrapAddrs {
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

func runScrapeRound(ctx context.Context, h host.Host, dht *dht.IpfsDHT) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	rlim := make(chan struct{}, 10)
	log.Info("scraping")
	scrapeRound := func(k string) {
		mctx, cancel := context.WithTimeout(ctx, keySearchTimeout)
		defer cancel()
		defer wg.Done()
		defer log.Infof("finished scraping new key")
		rlim <- struct{}{}
		defer func() {
			<-rlim
		}()

		peers, err := dht.GetClosestPeers(mctx, k)
		if err != nil {
			if mctx.Err() == nil {
				log.Error(err)
			}
			return
		}

		for {
			select {
			case _, ok := <-peers:
				if !ok {
					return
				}
			case <-mctx.Done():
				return
			}
		}
	}

	for i := 0; i < totalKeys; i++ {
		wg.Add(1)
		s, err := getRandomKey()
		if err != nil {
			return err
		}
		go scrapeRound(s)
	}
	wg.Wait()
	return nil
}

func getRandomKey() (string, error) {
	buf := make([]byte, 32)
	rand.Read(buf)
	o, err := mh.Encode(buf, mh.SHA2_256)
	if err != nil {
		return "", err
	}
	return string(o), nil
}