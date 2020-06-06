package main

import (
	"flag"
	"fmt"

	"github.com/alanshaw/libp2p-dht-scrape-aas/scraper"
	"github.com/alanshaw/libp2p-dht-scrape-aas/server"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("dht_scrape_aas")

func main() {
	logging.SetLogLevel("dht_scrape_aas", "info")

	port := flag.Int("port", 3000, "port to bind the API to")

	s, err := scraper.New()
	if err != nil {
		panic(err)
	}

	// s = scraper.NewAggregatingScraper(s)

	log.Infof("API listening on %s", fmt.Sprintf(":%v", *port))

	err = server.ListenAndServe(s, fmt.Sprintf(":%v", *port))
	if err != nil {
		panic(err)
	}
}
