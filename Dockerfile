FROM golang:1.14.2-buster

# Install deps
RUN apt-get update && apt-get install -y

WORKDIR /libp2p-dht-scrape-aas

COPY go.mod go.sum ./
RUN go mod download

# Copy the source from the current directory
# to the Working Directory inside the container
COPY . .

RUN go build -o libp2p-dht-scrape-aas .

# HTTP API
EXPOSE 3000

CMD ["./libp2p-dht-scrape-aas"]
