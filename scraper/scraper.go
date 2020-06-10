package scraper

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/alanshaw/libp2p-dht-scrape-aas/lp2p"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	mh "github.com/multiformats/go-multihash"
)

const (
	// totalRounds is the number of rounds performed by each host.
	totalRounds = 15
	// roundInterval is the sleep time between rounds.
	roundInterval = time.Second * 10
	// totalKeys is the number of random keys to get closest peers for, per round.
	totalKeys = 15
	// keySearchTimeout is the maximum time a closest peers query can run for.
	keySearchTimeout = time.Second * 30
	// peerStatChannelSize is the max number of buffered items in channels returned by Scrape.
	peerStatChannelSize = 100
	// peerUpdatedDebouncePeriod is the debounce period for when a peer is updated.
	// From testing:
	// 0 debounces (473 times) ready on average 0s
	// 1 debounces (420 times) ready on average 1.188270601s
	// 2 debounces (217 times) ready on average 2.626265233s
	// 3 debounces (195 times) ready on average 2.831655062s
	// 4 debounces (65 times) ready on average 6.235750843s
	peerUpdatedDebouncePeriod = time.Second * 3
)

var log = logging.Logger("dht_scrape_aas_scraper")

// PeerStat contains information about a seen peer.
type PeerStat struct {
	PeerID       string   `json:"peerID"`
	Addresses    []string `json:"addresses"`
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
	ch := make(chan PeerStat, peerStatChannelSize)

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

	peerUpdated := debouncePeerUpdated(ctx, func(pstore peerstore.Peerstore, peerID peer.ID) {
		var addrs []string
		for _, a := range pstore.Addrs(peerID) {
			addrs = append(addrs, a.String())
		}

		pstat := PeerStat{
			PeerID:    peerID.String(),
			Addresses: addrs,
		}

		av, err := pstore.Get(peerID, "AgentVersion")
		if err == nil {
			pstat.AgentVersion = fmt.Sprint(av)
		}

		protos, _ := pstore.GetProtocols(peerID)
		pstat.Protocols = protos

		select {
		case ch <- pstat:
		default:
			log.Warn("dropped peer stat due to full channel", pstat)
		}
	}, peerUpdatedDebouncePeriod)

	h, dht, err := lp2p.New(ctx, DefaultBootstrapAddrs, peerUpdated)
	if err != nil {
		return err
	}

	for i := 0; i < totalRounds; i++ {
		log.Infof("starting scrape round %d/%d", i+1, totalRounds)
		if err := runScrapeRound(ctx, h, dht); err != nil {
			return err
		}
		t := time.NewTimer(roundInterval)
		select {
		case <-t.C:
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		}
	}
	return nil
}

func debouncePeerUpdated(ctx context.Context, f lp2p.PeerUpdatedF, p time.Duration) lp2p.PeerUpdatedF {
	type peerUpdatedData struct {
		pstore      peerstore.Peerstore
		peerUpdated lp2p.PeerUpdatedF
	}

	l := sync.Mutex{}
	m := make(map[peer.ID]*peerUpdatedData)

	return func(pstore peerstore.Peerstore, peerID peer.ID) {
		l.Lock()
		defer l.Unlock()

		d, ok := m[peerID]
		if ok {
			d.peerUpdated = f
			return
		}

		t := time.NewTimer(p)
		d = &peerUpdatedData{pstore, f}
		m[peerID] = d

		go func() {
			select {
			case <-t.C:
				l.Lock()
				delete(m, peerID)
				l.Unlock()
				d.peerUpdated(d.pstore, peerID)
			case <-ctx.Done():
				t.Stop()
			}
		}()
	}
}

func runScrapeRound(ctx context.Context, h host.Host, dht *dht.IpfsDHT) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	rlim := make(chan struct{}, 10)
	scrapeRound := func(k string, i int) {
		mctx, cancel := context.WithTimeout(ctx, keySearchTimeout)
		defer cancel()
		defer wg.Done()
		defer log.Infof("scraped with key #%v", i)
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
		go scrapeRound(s, i)
	}
	wg.Wait()
	return ctx.Err()
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
