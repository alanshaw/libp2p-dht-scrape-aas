# libp2p DHT scrape aaS

A libp2p DHT scraper run as a service allowing anyone to collect, consume and use to generate useful data and visualisations.

The scraping implementation was heavily ~~inspired~~ stolen from [whyrusleeping/ipfs-counter](https://github.com/whyrusleeping/ipfs-counter).

## Install

No need to install, you can just request scrapings from the [HTTP API](#api). Send a `GET` request to `/peers` to receive an [ndjson](http://ndjson.org/) response that streams peers as they connect to the scraper.

## Usage

See the [API section](#api) below for details on HTTP API methods available.

For developers, simply clone this repo and start the scraper with `go run ./main.go`. The HTTP API is available by default at http://localhost:3000.

## API

### `GET /peers`

Returns an [ndjson](http://ndjson.org/) response that streams peers as they connect to the scraper. **Note**: you will see the same peer multiple times! The scraper creates many libp2p hosts with different peer IDs in order to survey a larger area of the DHT. It is stateless and only reports the information it discovers as it does so. It is up to you to collect and process the raw data.

Response objects look like:

```json
{
    "peerID": "QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtf3mvsqQLuvuJ",
    "addresses": ["/ip4/10.244.10.124/tcp/30800", "/ip4/159.65.73.69/tcp/30800"],
    "protocols": ["/ipfs/kad/1.0.0", "/libp2p/autonat/1.0.0", "/ipfs/bitswap/1.1.0"],
    "agentVersion": "go-ipfs/0.6.0-rc1/aa16952"
}
```

## Contribute

Feel free to dive in! [Open an issue](https://github.com/alanshaw/libp2p-dht-scrape-aas/issues/new) or submit PRs.

## License

[MIT](LICENSE) © Alan Shaw
