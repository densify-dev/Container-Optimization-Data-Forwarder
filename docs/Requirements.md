This section describes the candidate environment for the Container Optimization data collection.

The following, is required for Densify's Container Optimization data collection.
- Densify account, which is provided with a Densify subscription or through a free trial. See www.densify.com/service/signup. 
- Kubernetes or OpenShift 
  - Running cAdvisor as part of the kubelet that by default, provides the workload and configuration data required by Densify. 
- Prometheus
  - Provides monitoring/data aggregation layer. 
  - https://prometheus.io/
- kube-state-metrics
  - Requires version 1.5.0 or newer. 
  - Collects additional metrics from the Kubernetes API allowing Densify to get a complete picture of how the containers are setup. i.e. Replica Sets, Deployments, Pod and Container Labels.
  - https://github.com/kubernetes/kube-state-metrics.
The following item is not mandatory, but does provide additional environment information for Densify container optimization.
- Node Exporter
  - Collects data about the Nodes, on which the containers are running. 
  - https://hub.docker.com/r/prom/node-exporter/

Contact Support@Densify.com for more details.
