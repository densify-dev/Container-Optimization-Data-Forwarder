//Package cluster collects data related to containers and formats into csv files to send to Densify.
package cluster

import (
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//A node structure. Used for storing attributes and config details.
type clusterStruct struct {

	//Value fields
	cpuLimit, cpuRequest, memLimit, memRequest int
}

//Map that labels and values will be stored in
var clusterEntity = clusterStruct{}

//Hard-coded string for log file warnings
var entityKind = "Cluster"

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

	//Prefix for indexing (less clutter on screen)
	//var rsltIndex = result.(model.Matrix)

	/*
		==========START OF CLUSTER REQUEST/LIMIT METRICS==========
		-max(kube_pod_container_resource_limits_cpu_cores)*1000
		-avg(kube_pod_container_resource_requests_cpu_cores)*1000

		-max(kube_pod_container_resource_limits_memory_bytes)/1024/1024
		-avg(kube_pod_container_resource_requests_memory_bytes)/1024/1024
	*/

	query = `sum(kube_pod_container_resource_limits_cpu_cores)*1000`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "cpuLimit")
	getClusterMetric(result, "cpuLimit")

	query = `sum(kube_pod_container_resource_requests_cpu_cores)*1000`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "cpuRequest")
	getClusterMetric(result, "cpuRequest")

	query = `sum(kube_pod_container_resource_limits_memory_bytes)/1024/1024`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "memLimit")
	getClusterMetric(result, "memLimit")

	query = `avg(kube_pod_container_resource_requests_memory_bytes)/1024/1024`
	result = prometheus.MetricCollect(promaddress, query, start, end, entityKind, "memRequest")
	getClusterMetric(result, "memRequest")

	/*
		==========NODE REQUEST/LIMIT METRICS============
	*/

	writeAttributes(clusterName, promAddr)
	writeConfig(clusterName, promAddr)

	//Query and store prometheus CPU limit
	query = `kube_pod_container_resource_limits_cpu_cores*1000`
	getWorkload(promaddress, "cpu_limit", "CPU Limit", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "cpu_limit", "CPU Limit", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus Memory Limit
	query = `kube_pod_container_resource_requests_cpu_cores*1000`
	getWorkload(promaddress, "cpu_requests", "CPU Requests", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "cpu_requests", "CPU Requests", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus CPU limit
	query = `kube_pod_container_resource_limits_memory_bytes/1024/1024`
	getWorkload(promaddress, "memory_limit", "Memory Limit", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "memory_limit", "Memory Limit", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//Query and store prometheus Memory Limit
	query = `kube_pod_container_resource_requests_memory_bytes/1024/1024`
	getWorkload(promaddress, "memory_requests", "Memory Requests", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	getWorkload(promaddress, "memory_requests", "Memory Requests", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)
}
