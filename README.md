# libp2p DHT scrape aaS

A libp2p DHT scraper run as a service allowing anyone to collect, consume and use to generate useful reports and visualisations.

**https://dht.scrape.stream/peers**

The scraping implementation was heavily ~~inspired~~ stolen from [whyrusleeping/ipfs-counter](https://github.com/whyrusleeping/ipfs-counter).

This scraper works by first creating a libp2p host with a random peer ID. It then generates a random key and sends a `FIND_NODE` request in order to get a list of peers that are "closest" to the key. This is done muliple times and then the libp2p host is shutdown. A _new_ libp2p host with a random peer ID is then created and the cycle continues (...and it does not end).

The scraper hooks into the libp2p node's peerstore so that it can stream peer information as they are encountered. It exposes a HTTP endpoint where this information can be accessed.

**Note:** Peers may be encountered multiple times and may report _less information_ in a future encounter due to them being discovered but not connected to.

## Install

No need to install, you can just request scrapings from the [HTTP API](#api). Send a `GET` request to `/peers` to receive an [ndjson](http://ndjson.org/) response that streams peers as they are seen by the scraper.

## Usage

See the [API section](#api) below for details on HTTP API methods available.

### Developers

For developers, simply clone this repo and start the scraper with `go run ./main.go`. The HTTP API is available by default at http://localhost:3000.

#### Publish a new Docker image

```sh
# Build your container
docker build -t libp2p-dht-scrape-aas .

# Get it to run
docker run libp2p-dht-scrape-aas

# Commit new version
docker commit -m="some commit message" <CONTAINER_ID> alanshaw/libp2p-dht-scrape-aas

# Push to docker hub (must be logged in, do docker login)
docker push alanshaw/libp2p-dht-scrape-aas
```

## API

The public API is available at https://dht.scrape.stream

### `GET /peers`

Returns an [ndjson](http://ndjson.org/) response that streams peers as they are seen by the scraper. **Note**: you will see the same peer multiple times! The scraper creates many libp2p hosts with different peer IDs in order to survey a larger area of the DHT. It is stateless and only reports the information it discovers as it does so. It is up to you to collect and process the raw data.

Response objects look like:

```json
{
    "peerID": "QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtf3mvsqQLuvuJ",
    "addresses": ["/ip4/10.244.10.124/tcp/30800", "/ip4/159.65.73.69/tcp/30800"],
    "protocols": ["/ipfs/kad/1.0.0", "/libp2p/autonat/1.0.0", "/ipfs/bitswap/1.1.0"],
    "agentVersion": "go-ipfs/0.6.0-rc1/aa16952"
}
```

**Note:** `agentVersion` and `protocols` may be `""`/`[]` respectively if the scraper did not connect to the peer yet.

## Contribute

Feel free to dive in! [Open an issue](https://github.com/alanshaw/libp2p-dht-scrape-aas/issues/new) or submit PRs.

## License

[MIT](LICENSE) © Alan Shaw
