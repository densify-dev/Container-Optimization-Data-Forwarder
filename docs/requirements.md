This document describes the candidate environment for the Container Optimization Data Collection. At this point we are looking for a specific set of requirements in addition to ensuring that you are interested in container sizing or manifest optimization.

The following configuration is the typical configuration we have seen from participants in the beta program and would provide the best results for a beta trial:
- Kubernetes or OpenShift 
  - Running cAdvisor as part of the kubelet by default, provides the workload and configuration data required by Densify. 
- Prometheus
  - Provides monitoring/data aggregation layer. Documentation and sample configuration: https://devopscube.com/setup-prometheus-monitoring-on-kubernetes/
  - https://prometheus.io/
- kube-state-metrics
  - Collects additional metrics from the Kubernetes API allowing Densify to get a complete picture of how the containers are setup. ie Replica Sets, Deployments, Pod and Container Labels.
  - https://github.com/kubernetes/kube-state-metrics.
- Node Exporter
  - Collects data about the Nodes, on which the containers are running. 
  - https://hub.docker.com/r/prom/node-exporter/

Best results will be obtained if you are running a container environment as indicated above. If you are not using the typical configuration, then please answer the following questions to help us understand what configurations may be popular and to allow us to prioritize other options to support:
- If you are not using Docker, then please let us know what container runtime you are using?
- If you are not using Kubernetes (ex. Kubernetes, Docker Enterprise, or OpenShift) with kube-state-metrics, do you have cAdvisor installed? Are you running Docker Swarm?
- If you are not using Prometheus, then what tools do you use for collecting data and monitoring containers?  Would you consider installing Prometheus?
