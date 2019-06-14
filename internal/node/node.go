/*
This code is a prototype for node collection data for containers.

The skeleton query to group metrics by node and their values is (query made by Jack, H with help from Stephen, N):
	==============================
	max(max(label_replace(<METRIC GOES HERE>, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)
	==============================

	Things to look into:
		-See if you can calculate a min for workload
		-Update the CSV file names
		-Add taint query

	Last updated 10/06/2019
*/

//Package node collects data related to containers and formats into csv files to send to Densify.
package node

import (
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//A node structure. Used for storing attributes and config details. (Maybe delete the values?)
type node struct {
	node, nodeLabel                                                               string
	labelBetaKubernetesIoArch, labelBetaKubernetesIoOs, labelKubernetesIoHostname string

	//Value fields
	taint                                                                                                 int
	diskReadBytes, diskWriteBytes, activeMemBytes, memTotalBytes, netReceiveBytes                         int
	netReceivePackets, netSpeedBytes, netTransmitBytes, netTransmitPackets, cpuSecs                       int
	cpuCapacity, memCapacity, ephemeralStorageCapacity, podsCapacity, hugepages2MiCapacity                int
	cpuAllocatable, memAllocatable, ephemeralStorageAllocatable, podsAllocatable, hugepages2MiAllocatable int
}

//Map that labels and values will be stored in
var nodes = map[string]*node{}

//Metrics a global func for collecting node level metrics in prometheus
func Metrics(clusterName, promProtocol, promAddr, promPort, interval string, intervalSize, history int, debug bool, currentTime time.Time) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var promaddress, query string
	var result model.Value
	var start, end time.Time

	//Start and end time + the prometheus address used for querying
	start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
	promaddress = promProtocol + "://" + promAddr + ":" + promPort

	//Query and store kubernetes node information/labels
	query = "max(kube_node_labels) by (instance, label_beta_kubernetes_io_arch, label_beta_kubernetes_io_os, label_kubernetes_io_hostname, node)"
	result = prometheus.MetricCollect(promaddress, query, start, end)

	//Prefix for indexing (less clutter on screen)
	var rsltIndex = result.(model.Matrix)

	//If result is not nil then continue with extraction
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			nodes[string(rsltIndex[i].Metric["node"])] =
				&node{
					node:                      string(rsltIndex[i].Metric["node"]),
					labelBetaKubernetesIoArch: string(rsltIndex[i].Metric["label_beta_kubernetes_io_arch"]),
					labelBetaKubernetesIoOs:   string(rsltIndex[i].Metric["label_beta_kubernetes_io_os"]),
					labelKubernetesIoHostname: string(rsltIndex[i].Metric["label_kubernetes_io_hostname"])}
		}
	}

	//Additonal config/attribute queries
	query = `kube_node_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getNodeMetricString(result, "node", "nodeLabel")

	query = `max(max(label_replace(node_network_speed_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getNodeMetric(result, "namespace", "node", "netSpeedBytes")

	//Queries capacity of nodes
	query = `kube_node_status_capacity`
	result = prometheus.MetricCollect(promaddress, query, start, end)

	//If the regular capacity query returns nothing then fall back on the older queries
	if result.(model.Matrix).Len() == 0 {
		query = `kube_node_status_capacity_cpu_cores`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getNodeMetric(result, "namespace", "node", "capacity_cpu")

		query = `kube_node_status_capacity_memory_bytes`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getNodeMetric(result, "namespace", "node", "capacity_mem")

		query = `kube_node_status_capacity_pods`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getNodeMetric(result, "namespace", "node", "capacity_pod")

	} else {
		getNodeMetric(result, "namespace", "node", "capacity")
	}

	//Queries allocatable metrics of the nodes
	query = `kube_node_status_allocatable`
	result = prometheus.MetricCollect(promaddress, query, start, end)

	//If the regular allocatable query returns nothing then fall back on the older queries
	if result.(model.Matrix).Len() == 0 {
		query = `kube_node_status_allocatable_cpu_cores`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getNodeMetric(result, "namespace", "node", "allocatable_cpu")

		query = `kube_node_status_allocatable_memory_bytes`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getNodeMetric(result, "namespace", "node", "allocatable_mem")

		query = `kube_node_status_allocatable_pods`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getNodeMetric(result, "namespace", "node", "allocatable_pod")

	} else {
		getNodeMetric(result, "namespace", "node", "allocatable")
	}

	//Write config and attribute files
	writeConfig(clusterName, promAddr)
	writeAttributes(clusterName, promAddr)

	/*
		==========START OF DISK METRICS========
		-node_disk_written_bytes_total 		(MAX)
		-node_disk_written_bytes_total 		(AVG)

		-node_disk_read_bytes_total    		(MAX)
		-node_disk_read_bytes_total    		(AVG)

		-irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m])    		(MAX)
		-irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m])    		(AVG)
	*/

	//Query and store prometheus node disk write in bytes (max)
	query = `max(max(label_replace(node_disk_written_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_write_bytes", "Raw Disk Write Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node disk write in bytes (avg)
	query = `avg(avg(label_replace(node_disk_written_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_write_bytes", "Raw Disk Write Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node disk read in bytes (max)
	query = `max(max(label_replace(node_disk_read_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_read_bytes", "Raw Disk Read Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node disk read in bytes (avg)
	query = `avg(avg(label_replace(node_disk_read_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_read_bytes", "Raw Disk Read Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Not 100% sure if this should be disk ops -SN
	//Query and store prometheus total disk read uptime as a percentage (max)
	query = `max(max(label_replace(irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_read_ops", "Disk Read Operations", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Not 100% sure if this should be disk ops -SN
	//Query and store prometheus total disk read uptime as a percentage (avg)
	query = `avg(avg(label_replace(irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_read_ops", "Disk Read Operations", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total disk write uptime as a percentage (max)
	query = `max(max(label_replace(irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_write_ops", "Disk Write Operations", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total disk write uptime as a percentage (avg)
	query = `avg(avg(label_replace(irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
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

	//Would total be something we don't currently collect? -SN
	//Query and store prometheus node memory total in bytes (MAX)
	query = `max(max(label_replace(node_memory_MemTotal_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "memory_total_bytes", "Total Memory Bytes", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Would total be something we don't currently collect? -SN
	//Query and store prometheus node memory total in bytes (AVG)
	query = `avg(avg(label_replace(node_memory_MemTotal_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "memory_total_bytes", "Total Memory Bytes", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory active bytes (MAX)
	query = `max(max(label_replace(node_memory_Active_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "memory_active_bytes", "Active Memory Bytes", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory active bytes (AVG)
	query = `avg(avg(label_replace(node_memory_Active_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "memory_active_bytes", "Active Memory Bytes", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory total in bytes (MAX)
	query = `max(max(label_replace(node_memory_MemTotal_bytes - node_memory_MemFree_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "memory_raw_bytes", "Raw Memory Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory total in bytes (AVG)
	query = `avg(avg(label_replace(node_memory_MemTotal_bytes - node_memory_MemFree_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "memory_raw_bytes", "Raw Memory Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory total free in bytes (MAX)
	query = `max(max(label_replace(node_memory_MemTotal_bytes - (node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes), "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "memory_actual_workload", "Actual Memory Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Do we need to collect the AVG of this metric? -SN
	//Query and store prometheus node memory total free in bytes (AVG)
	query = `avg(avg(label_replace(node_memory_MemTotal_bytes - (node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes), "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
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

	//Query and store prometheus node recieved network data in bytes (MAX)
	query = `max(max(label_replace(node_network_receive_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_received_bytes", "Raw Net Received Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node recieved network data in bytes (AVG)
	query = `avg(avg(label_replace(node_network_receive_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_received_bytes", "Raw Net Received Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus recieved network data in packets (MAX)
	query = `max(max(label_replace(node_network_receive_packets_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_received_packets", "Network Packets Received", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus recieved network data in packets (AVG)
	query = `avg(avg(label_replace(node_network_receive_packets_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_received_packets", "Network Packets Received", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total transmitted network data in bytes (MAX)
	query = `max(max(label_replace(node_network_transmit_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_sent_bytes", "Raw Net Sent Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total transmitted network data in bytes (AVG)
	query = `avg(avg(label_replace(node_network_transmit_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_sent_bytes", "Raw Net Sent Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total transmitted network data in packets (MAX)
	query = `max(max(label_replace(node_network_transmit_packets_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_sent_packets", "Network Packets Sent", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total transmitted network data in packets (AVG)
	query = `avg(avg(label_replace(node_network_transmit_packets_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_sent_packets", "Network Packets Sent", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	/*
		==========END OF NETWORK METRICS============
	*/

	//**************************************************************************************************************
	//**************************************************************************************************************

	/*
		==========START OF CPU METRICS==========
		-rate(node_cpu_seconds_total{mode!="idle"}[5m])) by (pod, instance, cpu)*100	(MAX)
		-rate(node_cpu_seconds_total{mode!="idle"}[5m])) by (pod, instance, cpu)*100	(AVG)
	*/

	//Query and store prometheus total cpu uptime in seconds (MAX)
	query = `max(max(label_replace(sum(rate(node_cpu_seconds_total{mode!="idle"}[5m])) by (pod, instance, cpu)*100, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "cpu_utilization", "CPU Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total cpu uptime in seconds (AVG)
	query = `avg(avg(label_replace(sum(rate(node_cpu_seconds_total{mode!="idle"}[5m])) by (pod, instance, cpu)*100, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "cpu_utilization", "CPU Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	/*
		==========END OF CPU METRICS============
	*/
}
