/*
This code is a prototype for node collection data for containers.

The skeleton query to group metrics by node and their values is (query made by Jack, H with help from Stephen, N):
	==============================
	max(max(label_replace(<METRIC GOES HERE>, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)
	==============================

	Things to look into:
		-See if you can calculate a min for workload
		-Update the CSV file names
		-Add taint query

	Last updated 24/05/2019
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
	node, namespace, nodeLabel                                                    string
	labelBetaKubernetesIoArch, labelBetaKubernetesIoOs, labelKubernetesIoHostname string

	//Value fields
	taint, allocatable, capacity                                                    int
	diskReadBytes, diskWriteBytes, activeMemBytes, memTotalBytes, netReceiveBytes   int
	netReceivePackets, netSpeedBytes, netTransmitBytes, netTransmitPackets, cpuSecs int
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
	query = "max(kube_node_labels) by (instance, label_beta_kubernetes_io_arch, label_beta_kubernetes_io_os, label_kubernetes_io_hostname, node, namespace)"
	result = prometheus.MetricCollect(promaddress, query, start, end)

	//Prefix for indexing (less clutter on screen)
	var rsltIndex = result.(model.Matrix)

	//If result is not nil then continue with extraction
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			nodes[string(rsltIndex[i].Metric["node"])] =
				&node{
					node:                      string(rsltIndex[i].Metric["node"]),
					namespace:                 string(rsltIndex[i].Metric["namespace"]),
					labelBetaKubernetesIoArch: string(rsltIndex[i].Metric["label_beta_kubernetes_io_arch"]),
					labelBetaKubernetesIoOs:   string(rsltIndex[i].Metric["label_beta_kubernetes_io_os"]),
					labelKubernetesIoHostname: string(rsltIndex[i].Metric["label_kubernetes_io_hostname"])}
		}
	}

	//Additonal config/attribute queries
	query = `kube_node_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getNodeMetricString(result, "node", "nodeLabel")

	query = `max(max(label_replace(node_network_speed_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getNodeMetric(result, "namespace", "node", "netSpeedBytes")

	//Write config and attribute files
	writeConfig(promAddr)
	writeAttributes(promAddr)

	/*
		==========START OF DISK METRICS========
		-node_disk_written_bytes_total 		(MAX)
		-node_disk_written_bytes_total 		(AVG)

		-node_disk_read_bytes_total    		(MAX)
		-node_disk_read_bytes_total    		(AVG)

		-irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m])    		(MAX)
	*/

	//Query and store prometheus node disk write in bytes (max)
	query = `max(max(label_replace(node_disk_written_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_workload", "Disk Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node disk write in bytes (avg)
	query = `avg(avg(label_replace(node_disk_written_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_workload", "Disk Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node disk read in bytes (max)
	query = `max(max(label_replace(node_disk_read_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_read", "Disk Read", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node disk read in bytes (avg)
	query = `avg(avg(label_replace(node_disk_read_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_read", "Disk Read", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total disk read uptime (percentage)
	query = `max(max(label_replace(irate(node_disk_read_time_seconds_total[5m]) / irate(node_disk_io_time_seconds_total[5m]), "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "disk_read_time", "Disk read time", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

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

	//Query and store prometheus node memory total in bytes (MAX)
	query = `max(max(label_replace(node_memory_MemTotal_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "total_mem_bytes", "total_mem_bytes", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory total in bytes (AVG)
	query = `avg(avg(label_replace(node_memory_MemTotal_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "total_mem_bytes", "total_mem_bytes", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory active bytes (MAX)
	query = `max(max(label_replace(node_memory_Active_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "active_mem_bytes", "active_mem_bytes", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory active bytes (AVG)
	query = `avg(avg(label_replace(node_memory_Active_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "active_mem_bytes", "active_mem_bytes", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory total (in bytes) MAX
	query = `max(max(label_replace(node_memory_MemTotal_bytes - node_memory_MemFree_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "mem_raw_workload", "Raw Mem Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory total (in bytes) AVG
	query = `avg(avg(label_replace(node_memory_MemTotal_bytes - node_memory_MemFree_bytes, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "mem_raw_workload", "Raw Mem Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory total free (in bytes) MAX
	query = `max(max(label_replace(node_memory_MemTotal_bytes - (node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes), "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "mem_auctual_workload", "Auctual Mem Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node memory total free (in bytes) AVG IS THIS NEEDED?
	query = `avg(avg(label_replace(node_memory_MemTotal_bytes - (node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes), "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "mem_auctual_workload", "Auctual Mem Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

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
	query = `max(max(label_replace(node_network_receive_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_rec_bytes", "net_rec_bytes", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus node recieved network data in bytes (AVG)
	query = `avg(avg(label_replace(node_network_receive_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_rec_bytes", "net_rec_bytes", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus recieved network data in packets (MAX)
	query = `max(max(label_replace(node_network_receive_packets_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_rec_packets", "net_rec_packets", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus recieved network data in packets (AVG)
	query = `avg(avg(label_replace(node_network_receive_packets_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_rec_packets", "net_rec_packets", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total transmitted network data in bytes (MAX)
	query = `max(max(label_replace(node_network_transmit_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_transmit_bytes", "net_transmit_bytes", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total transmitted network data in bytes (AVG)
	query = `avg(avg(label_replace(node_network_transmit_bytes_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_transmit_bytes", "net_transmit_bytes", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total transmitted network data in packets (MAX)
	query = `max(max(label_replace(node_network_transmit_packets_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_transmit_packets", "total_transmit_packets", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total transmitted network data in packets (AVG)
	query = `max(max(label_replace(node_network_transmit_packets_total, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "net_transmit_packets", "total_transmit_packets", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

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
	query = `max(max(label_replace(sum(rate(node_cpu_seconds_total{mode!="idle"}[5m])) by (pod, instance, cpu)*100, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "cpu_time_secs", "cpu_time_secs", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus total cpu uptime in seconds (AVG)
	query = `avg(avg(label_replace(sum(rate(node_cpu_seconds_total{mode!="idle"}[5m])) by (pod, instance, cpu)*100, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~"node-exporter.*"}) by (node)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getWorkload(promaddress, "cpu_time_secs", "cpu_time_secs", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	/*
		==========END OF CPU METRICS============
	*/
}
