This section describes the prerequisites for Densify's Container Optimization data collection.

- Densify account. Contact Densify for details of your subscription or sign up for a free trial. See www.densify.com/service/signup. 
- Kubernetes or OpenShift must be deployed
  - Running cAdvisor as part of the kubelet that by default, provides the workload and configuration data required by Densify. 
- Prometheus
  - Provides monitoring/data aggregation layer. 
  - https://prometheus.io/
- kube-state-metrics
  - Requires version 1.5.0 or newer. 
  - The collected metrics allow Densify to get a complete picture of how your containers are setup. i.e. Replica Sets, Deployments, Pod and Container Labels.
  - https://github.com/kubernetes/kube-state-metrics.

The following items are not mandatory, but can provide additional environment information for Densify container optimization.
- Node Exporter
  - Collects data about the Nodes, on which the containers are running. 
  - https://hub.docker.com/r/prom/node-exporter/
- openshift-state-metrics
  - The collected metrics provide additional details for OpenShift-specific items such as Cluster Resource Quotas (CRQ).
  - https://github.com/openshift/openshift-state-metrics

Contact Support@Densify.com for more details.
