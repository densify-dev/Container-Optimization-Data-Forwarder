# Densify Container Data Forwarder

<img src="https://www.densify.com/wp-content/uploads/densify.png" width="300">

Densify Container Data Forwarder is the container that collects data from Kubernetes via Prometheus and forwards that data to Densify to analysis of how the containers are running to provide recommenditions about how to size the containers. 

- [Requirements](#requirements)
- [Usage](#usage)
- [Docker Images](#docker-images)
- [Examples](#examples)
- [Inputs](#inputs)
- [Outputs](#outputs)
- [License](#license)

## Requirements

- Densify account, which is provided with a Densify subscription or free trial (www.densify.com/trial)
- Kubernetes or OpenShift
- Prometheus (https://prometheus.io/)
- Kube-stste-metrics (https://github.com/kubernetes/kube-state-metrics)

## Usage

## Docker images

Docker image is available on [Docker Hub](https://hub.docker.com/r/kgillan/densify-kubernetes-data-forwarder)

To launch the container you will need to provide a configured config.cfg in the /config directory that provides details to connect to Prometheus and Densify servers. 

## Examples 

## Inputs

## Outputs

## License

Apache 2 Licensed. See LICENSE for full details.
