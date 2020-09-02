package version

const (
	// Version number of the DHT scrape node, it should be kept in sync with the current release tag.
	Version = "0.4.0"
	// UserAgent is the string passed by the identify protocol to other nodes in the network.
	UserAgent = "scrape-aas/" + Version
)
