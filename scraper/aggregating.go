package scraper

import (
	"context"
	"sync"
)

// aggregatingPeerStatChannelSize is the max number of buffered items in channels returned by Scrape.
const aggregatingPeerStatChannelSize = 500

type aggregatingScraper struct {
	Scraper
	aggregations struct {
		sync.Mutex
		M    map[*int]chan PeerStat
		done chan struct{}
	}
}

// NewAggregatingScraper wraps an existing scraper aggregating calls to Scrape and distributing them
func NewAggregatingScraper(s Scraper) Scraper {
	as := &aggregatingScraper{Scraper: s}
	as.aggregations.M = make(map[*int]chan PeerStat)
	as.aggregations.done = make(chan struct{})
	return as
}

func (as *aggregatingScraper) scrape() {
RESET:
	for {
		ctx, cancel := context.WithCancel(context.Background())
		ch := as.Scraper.Scrape(ctx)

		for {
			select {
			case pstat, ok := <-ch:
				if !ok {
					log.Warn("scraper closed channel")
					cancel()
					break RESET
				}
				as.aggregations.Lock()
				for _, c := range as.aggregations.M {
					select {
					case c <- pstat:
					default:
						log.Warn("aggregation dropped peer stat due to full channel", pstat)
					}
				}
				as.aggregations.Unlock()
			case <-as.aggregations.done:
				log.Info("aggregations done")
				cancel()
				return
			}
		}
	}
}

func (as *aggregatingScraper) Scrape(ctx context.Context) <-chan PeerStat {
	ch := make(chan PeerStat, aggregatingPeerStatChannelSize)
	var id int

	as.aggregations.Lock()
	as.aggregations.M[&id] = ch
	log.Infof("%v aggregations", len(as.aggregations.M))
	if len(as.aggregations.M) == 1 {
		go as.scrape()
	}
	as.aggregations.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			as.aggregations.Lock()
			close(as.aggregations.M[&id])
			delete(as.aggregations.M, &id)
			log.Infof("%v aggregations", len(as.aggregations.M))
			if len(as.aggregations.M) == 0 {
				as.aggregations.done <- struct{}{}
			}
			as.aggregations.Unlock()
		}
	}()

	return ch
}
