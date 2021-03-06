# Densify Container Optimization Data Forwarder

<img src="https://www.densify.com/wp-content/uploads/densify.png" width="300">

The Densify Container Optimization Data Forwarder is the container that collects data from Kubernetes via Prometheus and forwards that data to Densify. Densify then analyzes how the containers are running and provides sizing recommendations. 

- [Requirements](#requirements)
- [Docker Images](#docker-images)
- [Examples](#examples)
- [Documentation](#Documentation)
- [License](#license)

## Requirements

- Densify account, which is provided with a Densify subscription or through a free trial (https://www.densify.com/service/signup)
- Kubernetes or OpenShift
- Prometheus (https://prometheus.io/)
- Kube-state-metrics (https://github.com/kubernetes/kube-state-metrics)

## Docker images

The Docker image is available on [Docker Hub](https://hub.docker.com/r/densify/container-optimization-data-forwarder).

## Examples 
* [Kubernetes with Prometheus using Cron Job and Config Map](examples/CronJob)
* [Kubernetes with Authenticated Prometheus using Cron Job and Config Map](examples/AuthenticatedPrometheus)
* [Kubernetes with Prometheus using Config Map](examples/ConfigMap)
* [Docker with Volume Mount](examples/Docker)

## Documentation
* [Documentation](docs)

## License

Apache 2 Licensed. See [LICENSE](LICENSE) for full details.
