package lp2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	hook "github.com/alanshaw/ipfs-hookds"
	"github.com/alanshaw/libp2p-dht-scrape-aas/version"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	noise "github.com/libp2p/go-libp2p-noise"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	tls "github.com/libp2p/go-libp2p-tls"
	"github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	"github.com/multiformats/go-base32"
	"github.com/multiformats/go-multiaddr"
	secio "github.com/libp2p/go-libp2p-secio"
)

var (
	log       = logging.Logger("dht_scrape_aas_lp2p")
	peersRoot = ds.NewKey("/peers")
)

// PeerUpdatedF is a function called when a peer is updated in the peerstore.
type PeerUpdatedF func(peerstore.Peerstore, peer.ID)

// New creates a new libp2p host and DHT for use by a scraper.
func New(ctx context.Context, bootstrapAddrs []string, peerUpdated PeerUpdatedF) (host.Host, *dht.IpfsDHT, error) {
	var pstore peerstore.Peerstore
	afterPut := func(k ds.Key, v []byte, err error) error {
		peerID, _ := pstoreKeyToPeerID(k)
		if peerID != "" {
			go peerUpdated(pstore, peerID)
		}
		return err
	}
	pstoreDs := hook.NewBatching(dssync.MutexWrap(ds.NewMapDatastore()), hook.WithAfterPut(afterPut))

	pstore, err := pstoreds.NewPeerstore(ctx, pstoreDs, pstoreds.Options{
		CacheSize:           0,
		GCPurgeInterval:     0,
		GCLookaheadInterval: 0,
		GCInitialDelay:      60 * time.Second,
	})
	if err != nil {
		return nil, nil, err
	}

	h, err := libp2p.New(
		ctx,
		libp2p.UserAgent(version.UserAgent),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Peerstore(pstore),
		libp2p.Transport(quic.NewTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(ws.New),
		libp2p.Security(tls.ID, tls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(secio.ID, secio.New),
	)
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

var errInvalidKeyNamespaces = fmt.Errorf("not enough namespaces in peerstore record key")

// /peers/keys/CIQMTANQSIA5TBRC6KMBKUPIVFYZ6MOQNY4233JOXZ37FY52H7KW3YY/pub
// /peers/metadata/CIQB62TDWSJAVTIVR3Z3LXCZVGVOOL56IWEQUY5F2HX4QTSPLOQHEXA/protocols
// /peers/addrs/CIQB62TDWSJAVTIVR3Z3LXCZVGVOOL56IWEQUY5F2HX4QTSPLOQHEXA
// etc...
func pstoreKeyToPeerID(k ds.Key) (peer.ID, error) {
	nss := k.Namespaces()
	if len(nss) < 3 {
		return "", errInvalidKeyNamespaces
	}

	b, err := base32.RawStdEncoding.DecodeString(nss[2])
	if err != nil {
		return "", err
	}

	return peer.IDFromBytes(b)
}
