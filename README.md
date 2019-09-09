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
* [Kubernetes Cron Job with Config Map](examples/CronJob)
* [Kubernetes with Config Map](examples/ConfigMap)
* [Docker with Volume Mount](examples/Docker)

## Helm Chart

To deploy it via Helm follow these steps:
* Clone the repo
* Set the relevant endpoints and credentials values in helm/resources/overrideValues.yaml (see the configuration table below)
* cd helm
* Run the command: 'helm install . -f resources/overrideValues.yaml'

 ## Helm chart Configuration

The following table lists the configurable parameters of the container-optimization-data-forwarder chart.

| Parameter                                | Description                                             | Default                   |
|------------------------------------------|---------------------------------------------------------|---------------------------|
| `config.densify.hostname`                      | Host Name / IP of the Densify server                |           |
| `config.densify.port`                   | Port of the Densify server                             |                |
| `config.densify.protocol`            | Protocol for Densify server connectivity (http/https)           |                       |
| `config.densify.user`           | Username to access Densify server                 |                     |
| `config.densify.password`        | Passsord to access Densify server                                     |                   |
| `config.prometheus.hostname`      | Host Name / IP of the Prometheus server             |                           |
| `config.prometheus.port`               | Port to connect in Prometheus server                                    |   |
| `config.zipEnabled`                      |    Controls whether contents are zipped before transmission                                   | true              |
| `config.zipname`               | Name of the zip file that archives the content                                     |             |

## Documentation
* [Documentation](docs)

## License

Apache 2 Licensed. See [LICENSE](LICENSE) for full details.
