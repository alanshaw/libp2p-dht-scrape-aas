# libp2p DHT scrape aaS Kubenetes Configuration

These are the configuration files used to deploy the libp2p DHT scraper.

## Deploying to DigitalOcean

First create a cluster with one machine.

Next install [`doctl`](https://github.com/digitalocean/doctl) and [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) and run the following commands to deploy the scraper:

```sh
# Get k8s config and set it as the current context
doctl kubernetes cluster kubeconfig save <your_cluster_name>
# Create the namespace that the scraper runs in
kubectl create -f k8s/namespace.yaml
# Apply configs
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

## Updating a deployment

The config uses the latest `alanshaw/libp2p-dht-scrape-aas` image, so if you've tagged an pushed a new version all you need to do is scale down and up each deployment:

```sh
kubectl scale deployment/scraper-deployment --replicas=0 -n libp2p-dht-scrape-aas
kubectl scale deployment/scraper-deployment --replicas=1 -n libp2p-dht-scrape-aas
```