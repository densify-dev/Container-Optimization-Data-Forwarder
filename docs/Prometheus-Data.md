The following tables list Prometheus metrics and their usage.

- [Container Metrics](#container-metrics)
- [Node Metrics](#node-metrics)
- [Node Group Metrics](#node-group-metrics)
- [Cluster Metrics](#cluster-metrics)
- [CRQ Metrics](#crq-metrics)


## Container Metrics
| Metric | Description | JM-Comments | 
|--------|-------|-------|
| container_spec_cpu_shares | Container labels | is this really a label? |
| kube_pod_container_info | Container information | Why is it kube, pod and container? |
| container_cpu_usage_seconds_total | Container CPU utilization in mCores |
| kube_pod_container_resource_limits_cpu_cores | Container CPU limit | You have units above, but not for any other metrics. Maybe need a col for units |
| kube_pod_container_resource_requests_cpu_cores | Container CPU requests |
| container_memory_usage_bytes | Container raw memory utilization |
| container_memory_rss | Container actual memory utilization | Iss RSS actual usage? |
| container_spec_memory_limit_bytes | Container memory | what is a spec limit |
| kube_pod_container_resource_limit_memory_bytes | Container memory limit |
| kube_pod_container_resource_requests_memory_bytes | Container memory requests |
| container_fs_usage_bytes | Container raw disk utilization |  
| kube_pod_container_status_restarts_total | Container restarts | 
| kube_pod_container_status_terminated | Container power state | Is this power state or just state?|
| kube_pod_labels | Pod labels |
| kube_pod_info | Pod information |
| kube_pod_created | Pod creation time |
| kube_pod_owner | Pod owner |
| kube_namespace_labels | Namespace labels |
| kube_limitrange | Namespace limit |
| kube_replicaset_labels | ReplicaSet labels |
| kube_replicaset_created | ReplicaSet creation time |
| kube_replicaset_owner | ReplicaSet owner |
| kube_replicaset_spec_replicas | Replicaset current size & Deployment current size|
| kube_deployment_labels | Deployment labels |
| kube_deployment_created | Deployment creation time |
| kube_deployment_spec_strategy_rollingupdate_max_surge | Deployment max surge | should we add for "rolling update" |
| kube_deployment_spec_strategy_rollingupdate_max_unavailable | Deployment max unavailable | should we add "for rolling update" |
| kube_deployment_metadata_generation | Deployment metadata generation |
| kube_deployment_status_replicas_available | Deployment status of replicas available | should this be "of available replicas"? |
| kube_deployment_status_replicas | Deployment status replicas |  should this be "of replicas"? |
| kube_deployment_spec_replicas | Deployment spec replicas | should this be "for replicas"? |
| kube_job_labels | Job labels |
| kube_job_info | Job information |
| kube_job_created | Job creation time |
| kube_job_owner | Job owner |
| kube_job_spec_completions | Job spec completions | What is this telling you? |
| kube_job_spec_parallelism | Job spec parallelism, current size and CronJob curent size |
| kube_job_status_completion_time | Job status completion time |
| kube_job_status_start_time | Job status start time |
| kube_cronjob_labels | CronJob labels |
| kube_cronjob_info | CronJob information |
| kube_cronjob_created | CronJob creation time |
| kube_cronjob_next_schedule_time | CronJob next schedule time |
| kube_cronjob_status_last_schedule_time | CronJob last schedule time | 
| kube_cronjob_status_active | CronJob status active | What is this telling you? |
| kube_statefulset_labels | StatefulSet labels |
| kube_statefulset_created | StatefulSet creation time |
| kube_statefulset_replicas | StatefulSet current size |
| kube_daemonset_labels | DaemonSet labels |
| kube_daemonset_created | DaemonSet creation time |
| kube_daemonset_status_number_available | Daemonset current size |
| kube_replicationcontroller_created | Replication controller creation time |
| kube_replicationcontroller_spec_replicas | Replication controller current size |
| kube_hpa_labels | HPA labels |
| kube_hpa_spec_max_replicas | HPA max replicas |
| kube_hpa_spec_min_replicas | HPA min replicas |
| kube_hpa_status_condition | HPA scaling limited | should this be limit or limited to |
| kube_hpa_status_current_replicas | HPA current replicas |
| kube_hpa_status_desired_replicas | HPA desired replicas |

## Node Metrics
| Metric | Description | JM-Comments | 
|--------|-------|-------|
| kube_node_labels | Node labels |
| kube_node_info | Node info |
| node_network_speed_bytes | Network speed |
| kube_node_status_capacity | Node capacity |
| kube_node_status_capacity_cpu_cores | Node capacity CPU cores | should this be "in CPU cores"? |
| kube_node_status_capacity_memory_bytes | Node capacity memory bytes |  should this be "in memory bytes"? |
| kube_node_status_capacity_pods | Node capacity pods |  should this be "in number of pods"? |
| kube_node_status_allocatable | Node allocatable |  units? |
| kube_node_status_allocatable_cpu_cores | Node allocatable CPU cores |  should this be "in CPU cores"? |
| kube_node_status_allocatable_memory_bytes | Node allocatable memory bytes |  should this be "in memory bytes"? |
| kube_node_status_allocatable_pods | Node allocatable pods | should this be "in number of pods"? |
| node_disk_written_bytes_total | Disk write bytes total |
| node_disk_read_bytes_total | Disk read bytes total |
| node_disk_write_time_seconds_total | Disk write operations | should this be "per second"? |
| node_disk_read_time_seconds_total | Disk read operations | should this be "per second"? |
| node_disk_io_time_seconds_total | Disk write operations & Disk read operations | should this be "per second"? |
| node_memory_MemTotal_bytes | Total memory bytes & Raw memory utilization & Actual memory utilization | Is this sum total or 3 values? |
| node_memory_MemFree_bytes | Raw memory utilization & Actual memory utilization | Should this be total- used? |
| node_memory_Cached_bytes | Actual memory utilization | Should this be "Actual memory utilized for cache"? |
| node_memory_Buffrees_bytes | Actual memory utilization | should this be "node_memory_Buffers_bytes? Should this be "Actual memory utilized for the buffer"? |
| node_memory_Active_bytes | Actual memory bytes | Is this mem used or available? |
| node_network_recieve_bytes_total | Raw net received utilization | Should this be "network utilization in bytes received"? |
| node_network_recieve_packets_total | Network packets received | Should this be "network utilization in packets received"? |
| node_network_transmit_bytes_total | Raw net sent utilization |  Should this be "network utilization in bytes transmitted"? |
| node_network_transmit_packets_total | Network packets sent |  Should this be "network utilization in packets transmitted"? |
| node_cpu_seconds_total | CPU utilization | what is the sec for? |

## Node Group Metrics
| Metric | Description |  JM-Comments | 
|--------|-------| -------|
| kube_node_labels | Node Group labels |
| kube_pod_container_resource_limits_cpu_cores | CPU limit (used for workload and attribute) |
| kube_pod_container_resource_requests_cpu_cores | CPU requests (used for workload and attribute) |
| kube_pod_container_resource_limits_memory_bytes | Memory limit (used for workload and attribute) |
| kube_pod_container_resource_requests_memory_bytes | Memory requests (used for workload and attribute) |
| kube_node_status_capacity_cpu_cores | Node Group average capacity CPU cores | Should this be "capacity in CPU cores"? |
| kube_node_status_capacity_memory_bytes | Node Group average capacity memory bytes |  Should this be "capacity in memory bytes "? |
| node_disk_written_bytes_total | Disk write bytes total (avg) |
| node_disk_read_bytes_total | Disk read bytes total (avg) |
| node_disk_write_time_seconds_total | Disk write operations (avg) | per second? |
| node_disk_read_time_seconds_total | Disk read operations (avg) | per second? |
| node_disk_io_time_seconds_total | Disk write operations (avg) & Disk read operations (avg) | per second? |
| node_memory_MemTotal_bytes | Total memory bytes (avg) & Raw memory utilization (avg) & Actual memory utilization (avg) |
| node_memory_MemFree_bytes | Raw memory utilization (avg) & Actual memory utilization (avg) |  Is this sum total or 3 values? |
| node_memory_Cached_bytes | Actual memory utilization (avg) | Should this be "Actual memory utilized for cache"? |
| node_memory_Buffrees_bytes | Actual memory utilization (avg) | Is this metric "node_memory_Buffers_bytes? Should this be "Actual memory utilized for the buffer"? |
| node_memory_Active_bytes | Actual memory bytes (avg) |  Is this mem used or available? |
| node_network_recieve_bytes_total | Raw net received utilization (avg) | Should this be "network utilization in bytes received"? |
| node_network_recieve_packets_total | Network packets received (avg) |  Should this be "network utilization in packets received"? |
| node_network_transmit_bytes_total | Raw net sent utilization (avg) |  Should this be "network utilization in bytes transmitted"? |
| node_network_transmit_packets_total | Network packets sent (avg) |   Should this be "network utilization in packets transmitted"? |
| node_cpu_seconds_total | CPU utilization (avg) | what is the sec for? |

## Cluster Metrics

How are these metrics different from the node, above?

| Metric | Desription |
|--------|-------|
| kube_pod_container_resource_limits_cpu_cores | CPU limit (used for workload and attribute) |
| kube_pod_container_resource_requests_cpu_cores | CPU requests (used for workload and attribute) |
| kube_pod_container_resource_limits_memory_bytes | Memory limit (used for workload and attribute) |
| kube_pod_container_resource_requests_memory_bytes | Memory requests (used for workload and attribute) |
| kube_node_status_capacity_cpu_cores | Avg capacity CPU cores |
| kube_node_status_capacity_memory_bytes | Avg capacity memory bytes |
| node_disk_written_bytes_total | Disk write bytes total (avg) |
| node_disk_read_bytes_total | Disk read bytes total (avg) |
| node_disk_write_time_seconds_total | Disk write operations (avg) |
| node_disk_read_time_seconds_total | Disk read operations (avg) |
| node_disk_io_time_seconds_total | Disk write operations (avg) & Disk read operations (avg) |
| node_memory_MemTotal_bytes | Total memory bytes (avg) & Raw memory utilization (avg) & Actual memory utilization (avg) |
| node_memory_MemFree_bytes | Raw memory utilization (avg) & Actual memory utilization (avg) |
| node_memory_Cached_bytes | Actual memory utilization (avg) |
| node_memory_Buffrees_bytes | Actual memory utilization (avg) |
| node_memory_Active_bytes | Actual memory bytes (avg) |
| node_network_recieve_bytes_total | Raw net received utilization (avg) |
| node_network_recieve_packets_total | Network packets received (avg) |
| node_network_transmit_bytes_total | Raw net sent utilization (avg) |
| node_network_transmit_packets_total | Network packets sent (avg) |
| node_cpu_seconds_total | CPU utilization (avg) |

## CRQ Metrics
Available only for OpenShift.

| Metric | Usage |
|--------|-------|
| openshift_clusterresourcequota_created | Cluster Resource Quota creation time |
| openshift_clusterresourcequota_selector | Cluster Resource Quota information |
| openshift_clusterresourcequota_labels | Cluster Resource Quota labels |
| openshift_clusterresourcequota_usage | CPU\memory request\limit utlization (used for workload and attribute) |
| openshift_clusterresourcequota_namespace_usage | Namespace usage information |
