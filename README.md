# Densify Container Optimization Data Forwarder

<img src="https://www.densify.com/wp-content/uploads/densify.png" width="300">

The Densify Container Optimization Data Forwarder is the container that collects data from Kubernetes via Prometheus and forwards that data to Densify. Densify then analyzes how the containers are running and provides sizing recommendations. 

- [Requirements](#requirements)
- [Docker Images](#docker-images)
- [Examples](#examples)
- [Documentation](#Documentation)
- [License](#license)

## Requirements

- Densify account, which is provided with a Densify subscription or through a free trial (www.densify.com/trial)
- Kubernetes or OpenShift
- Prometheus (https://prometheus.io/)
- Kube-state-metrics (https://github.com/kubernetes/kube-state-metrics)

## Docker images

The Docker image is available on [Docker Hub](https://hub.docker.com/r/densify/container-optimization-data-forwarder).

## Examples 
* [Docker with Volume Mount](examples/Docker)
* [Kubernetes with Config Map](examples/ConfigMap)

## Documentation
* [Documentation](docs)

## License

Apache 2 Licensed. See [LICENSE](LICENSE) for full details.
