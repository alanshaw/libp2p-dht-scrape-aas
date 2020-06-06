package scraper

import (
	"context"
	"sync"
)

type aggregatingScraper struct {
	scraper      Scraper
	aggregations struct {
		sync.Mutex
		M map[*int]<-chan PeerStat
	}
}

// NewAggregatingScraper wraps an existing scraper aggregating calls to Scrape and distributing them
func NewAggregatingScraper(s Scraper) Scraper {
	as := &aggregatingScraper{scraper: s}
	as.aggregations.M = make(map[*int]<-chan PeerStat)
	return as
}

func (s *aggregatingScraper) Scrape(ctx context.Context) <-chan PeerStat {
	return nil
}
