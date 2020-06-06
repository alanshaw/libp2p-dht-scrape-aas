package server

import (
	"encoding/json"
	"net/http"

	"github.com/alanshaw/libp2p-dht-scrape-aas/scraper"
)

// ListenAndServe creates a new scraper HTTP server, and starts it up.
func ListenAndServe(s scraper.Scraper, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: NewServer(s),
	}
	return srv.ListenAndServe()
}

// NewServer creates a new scraper HTTP server
func NewServer(s scraper.Scraper) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/peers", peersHandler(s))
	return mux
}

func peersHandler(s scraper.Scraper) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		pstats := s.Scrape(r.Context())
		enc := json.NewEncoder(w)
		for pstat := range pstats {
			enc.Encode(pstat)
			f, ok := w.(http.Flusher)
			if ok {
				f.Flush()
			}
		}
	}
}
