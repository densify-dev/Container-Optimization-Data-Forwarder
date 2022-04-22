//Package node collects data related to containers and formats into csv files to send to Densify.
package node

import (
	"fmt"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

//A node structure. Used for storing attributes and config details.
type node struct {

	//Labels & general information about each node
	labelMap map[string]string

	//Value fields
	netSpeedBytes, cpuCapacity, memCapacity, ephemeralStorageCapacity, podsCapacity, hugepages2MiCapacity int
	cpuAllocatable, memAllocatable, ephemeralStorageAllocatable, podsAllocatable, hugepages2MiAllocatable int
	cpuLimit, cpuRequest, memLimit, memRequest                                                            int
}

//Map that labels and values will be stored in
var nodes = map[string]*node{}

//Hard-coded string for log file warnings
var entityKind = "node"

//Metrics a global func for collecting node level metrics in prometheus
func Metrics(args *common.Parameters) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var query string
	var result model.Value
	var haveNodeExport = true
	var err error

	//Start and end time + the prometheus address used for querying
	range5Min := common.TimeRange(args, historyInterval)

	//Query and store kubernetes node information/labels
	query = "max(kube_node_labels) by (instance, node)"
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.ErrorLogger.Println("metric=nodes query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=nodes query=" + query + " message=" + err.Error())
		return
	}
	var rsltIndex = result.(model.Matrix)
	for i := 0; i < rsltIndex.Len(); i++ {
		nodes[string(rsltIndex[i].Metric["node"])] =
			&node{
				labelMap: map[string]string{},

				//Network speed attribute (set to -1 by default to make error checking more easy)
				netSpeedBytes: -1,

				//Capacity and allocatable fields (set to -1 by default to make error checking more easy)
				cpuCapacity: -1, memCapacity: -1, ephemeralStorageCapacity: -1, podsCapacity: -1, hugepages2MiCapacity: -1,
				cpuAllocatable: -1, memAllocatable: -1, ephemeralStorageAllocatable: -1, podsAllocatable: -1, hugepages2MiAllocatable: -1,

				cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1}
	}

	//Additonal config/attribute queries
	query = `kube_node_labels`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=nodeLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeLabels query=" + query + " message=" + err.Error())
	} else {
		getNodeMetricString(result, "node")
	}

	//Additonal config/attribute queries
	query = `kube_node_info`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=nodeInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeInfo query=" + query + " message=" + err.Error())
	} else {
		getNodeMetricString(result, "node")
	}

	query = `kube_node_role`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=nodeInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeInfo query=" + query + " message=" + err.Error())
	} else {
		getNodeMetricString(result, "node")
	}

	//Gets the network speed in bytes as an attribute/config value for each node
	query = `label_replace(node_network_speed_bytes, "pod_ip", "$1", "instance", "(.*):.*")`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=networkSpeedBytes query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=networkSpeedBytes query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, "node", "netSpeedBytes")
	}

	if result.(model.Matrix).Len() == 0 {
		haveNodeExport = false
	}

	//Queries the capacity fields of all nodes
	query = `kube_node_status_capacity`
	result, err = common.MetricCollect(args, query, range5Min)

	/*
	  Some older versions of kube-state-metrics don't support kube_node_status_capacity.
	  If this is the case then we can use the older queries, which query the individual
	  metrics that kube_node_status_capacity returns.

	  NOTE: Not all queries from kube_node_status_capacity can be found in these
	  individual queries. If you see missing fields in the config/attribute files,
	  that is why.
	*/
	if result.(model.Matrix).Len() == 0 {
		//capacity_cpu_cores query
		query = `kube_node_status_capacity_cpu_cores`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=statusCapacityCpuCores query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=statusCapacityCpuCores query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "capacity_cpu")
		}

		//capacity_memory_bytes query
		query = `kube_node_status_capacity_memory_bytes`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=statusCapacityMemoryBytes query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=statusCapacityMemoryBytes query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "capacity_mem")
		}

		//capacity_pods query
		query = `kube_node_status_capacity_pods`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=statusCapacityPods query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=statusCapacityPods query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "capacity_pod")
		}

	} else {
		if err != nil {
			args.WarnLogger.Println("metric=statusCapacity query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=statusCapacity query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "capacity")
		}
	}

	//Queries the allocatable metric fields of all the nodes
	query = `kube_node_status_allocatable`
	result, err = common.MetricCollect(args, query, range5Min)

	/*
	  Some older versions of kube-state-metrics don't support kube_node_status_allocatable.
	  If this is the case then we can use the older queries, which query the individual
	  metrics that kube_node_status_allocatable returns.

	  NOTE: Not all queries from kube_node_status_allocatable can be found in these
	  individual queries. If you see missing fields in the config/attribute files,
	  that is why.
	*/
	if result.(model.Matrix).Len() == 0 {
		query = `kube_node_status_allocatable_cpu_cores`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=statusAllocatableCpuCores query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=statusAllocatableCpuCores query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "allocatable_cpu")
		}

		query = `kube_node_status_allocatable_memory_bytes`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=statusAllocatableMemoryBytes query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=statusAllocatableMemoryBytes query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "allocatable_mem")
		}

		query = `kube_node_status_allocatable_pods`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=statusAllocatablePods query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=statusAllocatablePods query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "allocatable_pod")
		}

	} else {
		if err != nil {
			args.WarnLogger.Println("metric=statusAllocatable query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=statusAllocatable query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "allocatable")
		}
	}

	query = `sum(kube_pod_container_resource_limits) by (node, resource)`
	result, err = common.MetricCollect(args, query, range5Min)
	if result.(model.Matrix).Len() == 0 {
		query = `sum(kube_pod_container_resource_limits_cpu_cores) by (node)*1000`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=cpuLimit query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cpuLimit query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "cpuLimit")
		}
		query = `sum(kube_pod_container_resource_limits_memory_bytes) by (node)/1024/1024`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=memLimit query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=memLimit query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "memLimit")
		}
	} else {
		getNodeMetric(result, "node", "limits")
	}

	query = `sum(kube_pod_container_resource_requests) by (node,resource)`
	result, err = common.MetricCollect(args, query, range5Min)
	if result.(model.Matrix).Len() == 0 {
		query = `sum(kube_pod_container_resource_requests_cpu_cores) by (node)*1000`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=cpuRequest query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cpuRequest query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "cpuRequest")
		}

		query = `sum(kube_pod_container_resource_requests_memory_bytes) by (node)/1024/1024`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=memRequest query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=memRequest query=" + query + " message=" + err.Error())
		} else {
			getNodeMetric(result, "node", "memRequest")
		}
	} else {
		getNodeMetric(result, "node", "requests")
	}

	//Writes the config and attribute files
	writeConfig(args)
	writeAttributes(args)

	//Checks to see if Node Exporter is installed. Based off if anything is returned from network speed bytes
	if haveNodeExport == false {
		args.ErrorLogger.Println("entity=" + entityKind + " message=It appears you do not have Node Exporter installed.")
		fmt.Println("[ERROR] entity=" + entityKind + " message=It appears you do not have Node Exporter installed.")
		return
	}

	var metricField []model.LabelName
	queryPrefix := ``
	queryPrefixSum := `sum(`
	querySuffix := ``
	querySuffixSum := `) by (instance)`
	metricField = append(metricField, "instance")

	//Check to see which disk queries to use if instance is IP address that need to link to pod to get name or if instance = node name.
	query = `max(max(label_replace(sum(irate(node_cpu_seconds_total{mode!="idle"}[` + args.SampleRateString + `m])) by (instance) / on (instance) group_left count(node_cpu_seconds_total{mode="idle"}) by (instance) *100, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result, err = common.MetricCollect(args, query, range5Min)

	if result.(model.Matrix).Len() != 0 {
		queryPrefix = `max(max(label_replace(`
		queryPrefixSum = `max(sum(label_replace(`
		querySuffix = `, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
		querySuffixSum = querySuffix
		metricField[0] = "node"
	}
	//Query and store prometheus total cpu uptime in seconds
	query = queryPrefix + `sum(irate(node_cpu_seconds_total{mode!="idle"}[` + args.SampleRateString + `m])) by (instance) / on (instance) group_left count(node_cpu_seconds_total{mode="idle"}) by (instance) *100` + querySuffix
	common.GetWorkload("cpu_utilization", "CpuUtilization", query, metricField, args, entityKind)

	//Query and store prometheus node memory total in bytes
	query = queryPrefix + `node_memory_MemTotal_bytes - node_memory_MemFree_bytes` + querySuffix
	common.GetWorkload("memory_raw_bytes", "MemoryBytes", query, metricField, args, entityKind)

	//Query and store prometheus node memory total free in bytes
	query = queryPrefix + `node_memory_MemTotal_bytes - (node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes)` + querySuffix
	common.GetWorkload("memory_actual_workload", "MemoryActualWorkload", query, metricField, args, entityKind)

	//Query and store prometheus node disk write in bytes
	query = queryPrefixSum + `irate(node_disk_written_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_write_bytes", "DiskWriteBytes", query, metricField, args, entityKind)

	//Query and store prometheus node disk read in bytes
	query = queryPrefixSum + `irate(node_disk_read_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_read_bytes", "DiskReadBytes", query, metricField, args, entityKind)

	//Query and store prometheus total disk read uptime as a percentage
	query = queryPrefixSum + `irate(node_disk_read_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_read_ops", "DiskReadOps", query, metricField, args, entityKind)

	//Query and store prometheus total disk write uptime as a percentage
	query = queryPrefixSum + `irate(node_disk_write_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_write_ops", "DiskWriteOps", query, metricField, args, entityKind)

	//Total disk values
	//Query and store prometheus node disk read in bytes
	query = queryPrefixSum + `irate(node_disk_read_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) + irate(node_disk_written_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_total_bytes", "DiskTotalBytes", query, metricField, args, entityKind)

	//Query and store prometheus total disk read uptime as a percentage
	query = queryPrefixSum + `(irate(node_disk_read_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) + irate(node_disk_write_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_total_ops", "DiskTotalOps", query, metricField, args, entityKind)

	//Query and store prometheus node recieved network data in bytes
	query = queryPrefixSum + `irate(node_network_receive_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_received_bytes", "NetReceivedBytes", query, metricField, args, entityKind)

	//Query and store prometheus recieved network data in packets
	query = queryPrefixSum + `irate(node_network_receive_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_received_packets", "NetReceivedPackets", query, metricField, args, entityKind)

	//Query and store prometheus total transmitted network data in bytes
	query = queryPrefixSum + `irate(node_network_transmit_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_sent_bytes", "NetSentBytes", query, metricField, args, entityKind)

	//Query and store prometheus total transmitted network data in packets
	query = queryPrefixSum + `irate(node_network_transmit_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_sent_packets", "NetSentPackets", query, metricField, args, entityKind)

	//Total values network
	//Query and store prometheus total network data in bytes
	query = queryPrefixSum + `irate(node_network_transmit_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m]) + irate(node_network_receive_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_total_bytes", "NetTotalBytes", query, metricField, args, entityKind)

	//Query and store prometheus total network data in packets
	query = queryPrefixSum + `irate(node_network_transmit_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m]) + irate(node_network_receive_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_total_packets", "NetTotalPackets", query, metricField, args, entityKind)

}
