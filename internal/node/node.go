//Package node collects data related to containers and formats into csv files to send to Densify.
package node

import (
	"log"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//A node structure. Used for storing attributes and config details.
type node struct {

	//Labels & general information about each node
	node, nodeLabel                                                               string
	labelBetaKubernetesIoArch, labelBetaKubernetesIoOs, labelKubernetesIoHostname string

	//Value fields
	netSpeedBytes, cpuCapacity, memCapacity, ephemeralStorageCapacity, podsCapacity, hugepages2MiCapacity int
	cpuAllocatable, memAllocatable, ephemeralStorageAllocatable, podsAllocatable, hugepages2MiAllocatable int
	cpuLimit, cpuRequest, memLimit, memRequest                                                            int
}

//Map that labels and values will be stored in
var nodes = map[string]*node{}

//Hard-coded string for log file warnings
var entityKind = "Node"

//Metrics a global func for collecting node level metrics in prometheus
func Metrics(clusterName, promProtocol, promAddr, promPort, interval string, intervalSize, history int, debug bool, currentTime time.Time) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var promaddress, query string
	var result model.Value
	var start, end time.Time
	var haveNodeExport = true

	//Start and end time + the prometheus address used for querying
	start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
	promaddress = promProtocol + "://" + promAddr + ":" + promPort

	//Query and store kubernetes node information/labels
	query = "max(kube_node_labels) by (instance, label_beta_kubernetes_io_arch, label_beta_kubernetes_io_os, label_kubernetes_io_hostname, node)"
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "nodeLabels")

	//Prefix for indexing (less clutter on screen)
	var rsltIndex = result.(model.Matrix)

	//If result is not nil then continue with extraction
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			nodes[string(rsltIndex[i].Metric["node"])] =
				&node{
					//String labels for node
					node:                      string(rsltIndex[i].Metric["node"]),
					labelBetaKubernetesIoArch: string(rsltIndex[i].Metric["label_beta_kubernetes_io_arch"]),
					labelBetaKubernetesIoOs:   string(rsltIndex[i].Metric["label_beta_kubernetes_io_os"]),
					labelKubernetesIoHostname: string(rsltIndex[i].Metric["label_kubernetes_io_hostname"]),
					nodeLabel:                 "",

					//Network speed attribute (set to -1 by default to make error checking more easy)
					netSpeedBytes: -1,

					//Capacity and allocatable fields (set to -1 by default to make error checking more easy)
					cpuCapacity: -1, memCapacity: -1, ephemeralStorageCapacity: -1, podsCapacity: -1, hugepages2MiCapacity: -1,
					cpuAllocatable: -1, memAllocatable: -1, ephemeralStorageAllocatable: -1, podsAllocatable: -1, hugepages2MiAllocatable: -1,

					cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1}
		}
	}

	//Additonal config/attribute queries
	query = `kube_node_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "nodeLabels")
	getNodeMetricString(result, "node", "nodeLabel")

	//Gets the network speed in bytes as an attribute/config value for each node
	query = `label_replace(node_network_speed_bytes, "pod_ip", "$1", "instance", "(.*):.*")`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "networkSpeedBytes")
	getNodeMetric(result, "node", "netSpeedBytes")

	if result.(model.Matrix).Len() == 0 {
		haveNodeExport = false
	}

	//Queries the capacity fields of all nodes
	query = `kube_node_status_capacity`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "statusCapacity")

	/*
	  Some older versions of prometheus don't support kube_node_status_capacity.
	  If this is the case then we can use the older queries, which query the individual
	  metrics that kube_node_status_capacity returns.

	  NOTE: Not all queries from kube_node_status_capacity can be found in these
	  individual queries. If you see missing fields in the config/attribute files,
	  that is why.
	*/
	if result.(model.Matrix).Len() == 0 {
		//capacity_cpu_cores query
		query = `kube_node_status_capacity_cpu_cores`
		result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "statusCapacityCpuCores")
		getNodeMetric(result, "node", "capacity_cpu")

		//capacity_memory_bytes query
		query = `kube_node_status_capacity_memory_bytes`
		result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "statusCapacityMemoryBytes")
		getNodeMetric(result, "node", "capacity_mem")

		//capacity_pods query
		query = `kube_node_status_capacity_pods`
		result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "statusCapacityPods")
		getNodeMetric(result, "node", "capacity_pod")

	} else {
		getNodeMetric(result, "node", "capacity")
	}

	//Queries the allocatable metric fields of all the nodes
	query = `kube_node_status_allocatable`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "statusAllocatable")

	/*
	  Some older versions of prometheus don't support kube_node_status_allocatable.
	  If this is the case then we can use the older queries, which query the individual
	  metrics that kube_node_status_allocatable returns.

	  NOTE: Not all queries from kube_node_status_allocatable can be found in these
	  individual queries. If you see missing fields in the config/attribute files,
	  that is why.
	*/
	if result.(model.Matrix).Len() == 0 {
		query = `kube_node_status_allocatable_cpu_cores`
		result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "statusAllocatableCpuCores")
		getNodeMetric(result, "node", "allocatable_cpu")

		query = `kube_node_status_allocatable_memory_bytes`
		result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "statusAllocatableMemoryBytes")
		getNodeMetric(result, "node", "allocatable_mem")

		query = `kube_node_status_allocatable_pods`
		result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "statusAllocatablePods")
		getNodeMetric(result, "node", "allocatable_pod")

	} else {
		getNodeMetric(result, "node", "allocatable")
	}

	//**************************************************************************************************************
	//**************************************************************************************************************

	/*
		==========START OF NODE REQUEST/LIMIT METRICS==========
		-sum(kube_pod_container_resource_limits_cpu_cores) by (node)*1000
		-sum(kube_pod_container_resource_requests_cpu_cores) by (node)*1000

		-sum(kube_pod_container_resource_limits_memory_bytes) by (node)/1024/1024
		-sum(kube_pod_container_resource_requests_memory_bytes) by (node)/1024/1024
	*/

	query = `sum(kube_pod_container_resource_limits_cpu_cores) by (node)*1000`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "cpuLimit")
	getNodeMetric(result, "node", "cpuLimit")

	query = `sum(kube_pod_container_resource_requests_cpu_cores) by (node)*1000`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "cpuRequest")
	getNodeMetric(result, "node", "cpuRequest")

	query = `sum(kube_pod_container_resource_limits_memory_bytes) by (node)/1024/1024`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "memLimit")
	getNodeMetric(result, "node", "memLimit")

	query = `sum(kube_pod_container_resource_requests_memory_bytes) by (node)/1024/1024`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "memRequest")
	getNodeMetric(result, "node", "memRequest")

	/*
		==========NODE REQUEST/LIMIT METRICS============
	*/

	//Writes the config and attribute files
	writeConfig(clusterName, promAddr)
	writeAttributes(clusterName, promAddr)

	//Checks to see if Node Exporter is installed. Based off if anything is returned from network speed bytes
	if haveNodeExport == false {
		log.Println(prometheus.LogMessage("[ERROR]", promaddress, entityKind, "N/A", "It appears you do not have Node Exporter installed.", "N/A"))
		return
	}

	/*
		==========START OF DISK METRICS========
		-node_disk_written_bytes_total 		(MAX)
		-node_disk_written_bytes_total 		(AVG)

		-node_disk_read_bytes_total    		(MAX)
		-node_disk_read_bytes_total    		(AVG)

		-irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m])    		(MAX)
		-irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m])    		(AVG)

		-irate(node_disk_write_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m]			(MAX)
		-irate(node_disk_write_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m]			(AVG)
	*/

	//Query and store prometheus node disk write in bytes
	query = `label_replace(irate(node_disk_written_bytes_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "disk_write_bytes", "Raw Disk Write Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "disk_write_bytes", "Raw Disk Write Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node disk read in bytes
	query = `label_replace(irate(node_disk_read_bytes_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "disk_read_bytes", "Raw Disk Read Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "disk_read_bytes", "Raw Disk Read Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total disk read uptime as a percentage
	query = `label_replace(irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "disk_read_ops", "Disk Read Operations", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "disk_read_ops", "Disk Read Operations", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total disk write uptime as a percentage
	query = `label_replace(irate(node_disk_write_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "disk_write_ops", "Disk Write Operations", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "disk_write_ops", "Disk Write Operations", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	/*
		==========END OF DISK METRICS==========
	*/

	//**************************************************************************************************************
	//**************************************************************************************************************

	/*
		==========START OF MEMORY METRICS==========
		-node_memory_MemTotal_bytes 		(MAX)
		-node_memory_MemTotal_bytes 		(AVG)

		-node_memory_Active_bytes			(MAX)
		-node_memory_Active_bytes			(AVG)

		-(node_memory_MemTotal_bytes - node_memory_MemFree_bytes)			(MAX)
		-(node_memory_MemTotal_bytes - node_memory_MemFree_bytes)			(AVG)

		-node_memory_MemTotal_bytes - (node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes)	(MAX)
		-node_memory_MemTotal_bytes - (node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes)	(AVG)
	*/

	//Query and store prometheus node memory total in bytes
	query = `label_replace(irate(node_memory_MemTotal_bytes[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "memory_total_bytes", "Total Memory Bytes", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "memory_total_bytes", "Total Memory Bytes", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory active bytes
	query = `label_replace(irate(node_memory_Active_bytes[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "memory_active_bytes", "Active Memory Bytes", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "memory_active_bytes", "Active Memory Bytes", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory total in bytes
	query = `label_replace(irate(node_memory_MemTotal_bytes[5m]) - irate(node_memory_MemFree_bytes[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "memory_raw_bytes", "Raw Memory Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "memory_raw_bytes", "Raw Memory Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory total free in bytes
	query = `label_replace(irate(node_memory_MemTotal_bytes[5m]) - (irate(node_memory_MemFree_bytes[5m]) + irate(node_memory_Cached_bytes[5m]) + irate(node_memory_Buffers_bytes[5m])), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "memory_actual_workload", "Actual Memory Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "memory_actual_workload", "Actual Memory Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	/*
		==========END OF MEMORY METRICS============
	*/

	//**************************************************************************************************************
	//**************************************************************************************************************

	/*
		==========START OF NETWORK METRICS==========
		-node_network_receive_bytes_total			(MAX)
		-node_network_receive_bytes_total			(AVG)

		-node_network_receive_packets_total			(MAX)
		-node_network_receive_packets_total			(AVG)

		-node_network_transmit_bytes_total			(MAX)
		-node_network_transmit_bytes_total			(AVG)

		-node_network_transmit_packets_total		(MAX)
		-node_network_transmit_packets_total		(AVG)
	*/

	//Query and store prometheus node recieved network data in bytes
	query = `label_replace(irate(node_network_receive_bytes_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "net_received_bytes", "Raw Net Received Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "net_received_bytes", "Raw Net Received Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus recieved network data in packets
	query = `label_replace(irate(node_network_receive_packets_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "net_received_packets", "Network Packets Received", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "net_received_packets", "Network Packets Received", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total transmitted network data in bytes
	query = `label_replace(irate(node_network_transmit_bytes_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "net_sent_bytes", "Raw Net Sent Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "net_sent_bytes", "Raw Net Sent Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total transmitted network data in packets
	query = `label_replace(irate(node_network_transmit_packets_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "net_sent_packets", "Network Packets Sent", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "net_sent_packets", "Network Packets Sent", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	/*
		==========END OF NETWORK METRICS============
	*/

	//**************************************************************************************************************
	//**************************************************************************************************************

	/*
		==========START OF CPU METRICS==========
		-irate(node_cpu_seconds_total{mode!="idle"}[5m])) by (pod, instance, cpu)*100	(MAX)
		-irate(node_cpu_seconds_total{mode!="idle"}[5m])) by (pod, instance, cpu)*100	(AVG)
	*/

	//Query and store prometheus total cpu uptime in seconds
	query = `label_replace(sum(irate(node_cpu_seconds_total{mode!="idle"}[5m])) by (pod, instance, cpu)*100, "pod_ip", "$1", "instance", "(.*):.*")`
	getWorkload(promaddress, "cpu_utilization", "CPU Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "cpu_utilization", "CPU Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	/*
		==========END OF CPU METRICS============
	*/

}
