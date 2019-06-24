A list of all prometheus metrics and their uses

## Container Metrics
| Metric | Usage | 
|--------|-------|
| container_spec_cpu_shares | Container labels |
| kube_pod_container_info | Container information |
| container_cpu_usage_seconds_total | Container CPU utilization in mCores |
| kube_pod_container_resource_limits_cpu_cores | Container CPU limit |
| kube_pod_container_resource_requests_cpu_cores | Container CPU requests |
| container_memory_usage_bytes | Container raw memory utilization |
| container_memory_rss | Container actual memory utilization |
| container_spec_memory_limit_bytes | Container memory |
| kube_pod_container_resource_limit_memory_bytes | Container memory limit |
| kube_pod_container_resource_requests_memory_bytes | Container memory requests |
| container_fs_usage_bytes | Container raw disk utilization |
| kube_pod_container_status_restarts_total | Container restarts |
| kube_pod_container_status_terminated | Container power state |
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
| kube_deployment_spec_strategy_rollingupdate_max_surge | Deployment max surge |
| kube_deployment_spec_strategy_rollingupdate_max_unavailable | Deployment max unavailable | 
| kube_deployment_metadata_generation | Deployment meta data generation |
| kube_deployment_status_replicas_available | Deployment status replicas available |
| kube_deployment_status_replicas | Deployment status replicas |
| kube_deployment_spec_replicas | Deployment spec replicas |
| kube_job_labels | Job labels |
| kube_job_info | Job information |
| kube_job_created | Job creation time |
| kube_job_owner | Job owner |
| kube_job_spec_completions | Job spec completions | 
| kube_job_spec_parallelism | Job spec parallelism & Job current size & CronJob curent size |
| kube_job_status_completion_time | Job status completion time |
| kube_job_status_start_time | Job status start time |
| kube_cronjob_labels | CronJob labels |
| kube_cronjob_info | CronJob information |
| kube_cronjob_created | CronJob creation time |
| kube_cronjob_next_schedule_time | CronJob next schedule time |
| kube_cronjob_status_last_schedule_time | CronJob last schedule time | 
| kube_cronjob_status_active | CronJob status active |
| kube_statefulset_labels | StatefulSet labels |
| kube_statefulset_created | StatefulSet creation time |
| kube_statefulset_replicas | StatefulSet current size |
| kube_daemonset_labels | DaemonSet labels |
| kube_daemonset_created | DaemonSet creation time |
| kube_daemonset_status_number_available | Daemonset current size |
| kube_replicationcontroller_created | Replication Controller creation time |
| kube_replicationcontroller_spec_replicas | Replication Controller current size |
| kube_hpa_labels | HPA labels |
| kube_hpa_spec_max_replicas | HPA max replicas |
| kube_hpa_spec_min_replicas | HPA min replicas |
| kube_hpa_status_condition | HPA scaling limited |
| kube_hpa_status_current_replicas | HPA current replicas |
| kube_hpa_status_desired_replicas | HPA desired replicas |

## Node Metrics
| Metric | Usage |
|--------|-------|
| kube_node_labels | Node labels |
| node_network_speed_bytes | Network speed |
| kube_node_status_capacity | Node capacity |
| kube_node_status_capacity_cpu_cores | Node capacity CPU cores |
| kube_node_status_capacity_memory_bytes | Node capacity memory bytes |
| kube_node_status_capacity_pods | Node capacity pods |
| kube_node_status_allocatable | Node allocatable |
| kube_node_status_allocatable_cpu_cores | Node allocatable CPU cores |
| kube_node_status_allocatable_memory_bytes | Node allocatable memory bytes |
| kube_node_status_allocatable_pods | Node allocatable pods |
| node_disk_written_bytes_total | Disk write bytes total (avg & max) |
| node_disk_read_bytes_total | Disk read bytes total (avg & max) |
| node_disk_read_time_seconds_total | Disk read operations (avg & max) |
| node_disk_io_time_seconds_total | Disk read operations (avg & max) |
| node_memory_MemTotal_bytes | Total memory bytes (avg & max) & Raw memory utilization (avg & max) & Actual memory utilization (avg & max) |
| node_memory_MemFree_bytes | Raw memory utilization (avg & max) & Actual memory utilization (avg & max) |
| node_memory_Cached_bytes | Actual memory utilization (avg & max) |
| node_memory_Buffrees_bytes | Actual memory utilization (avg & max) |
| node_memory_Active_bytes | Actual memory bytes (avg & max) |
| node_network_recieve_bytes_total | Recieved network bytes (avg & max) |
| node_network_recieve_packets_total | Recieved network packets (avg & max) |
| node_network_transmit_bytes_total | Raw net sent utilization (avg & max) |
| node_network_transmit_packets_total | Network packets sent (avg & max) |
| node_cpu_seconds_total | CPU utilization (avg & max) |
