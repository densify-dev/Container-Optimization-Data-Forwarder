# Densify Container Optimization Data Forwarder

<img src="https://www.densify.com/wp-content/uploads/densify.png" width="300">

The Densify Container Optimization Data Forwarder is the container that collects data from Kubernetes via Prometheus and forwards that data to Densify. Densify then analyzes how the containers are running and provides sizing recommendations. 

- [Requirements](#requirements)
- [Docker Images](#docker-images)
- [License](#license)

## Requirements

- Densify account, which is provided with a Densify subscription or through a free trial (www.densify.com/trial)
- Kubernetes or OpenShift
- Prometheus (https://prometheus.io/)
- Kube-state-metrics (https://github.com/kubernetes/kube-state-metrics)

## Docker images

The Docker image is available on [Docker Hub](https://hub.docker.com/r/densify/container-optimization-data-forwarder).

To launch the container do the following:
1. Update the [config.cfg](https://github.com/densify-dev/Container-Optimization-Data-Forwarder/blob/master/trans/config/config.cfg) file located in the /config directory. This file provides details to connect to both the Prometheus and Densify servers.

2. Execute the following command to create and run the data forwarder, connect to Prometheus and then to Densify:
```bash
docker run -v "/config/config.cfg":"/config/config.cfg" \
  densify/container-optimization-data-forwarder
```
Densify then loads the collected data.

## License

Apache 2 Licensed. See [LICENSE](https://github.com/densify-dev/Container-Optimization-Data-Forwarder/blob/master/LICENSE) for full details.
