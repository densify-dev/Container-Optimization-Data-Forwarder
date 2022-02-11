//Package node collects data related to nodes and formats into json files to send to Densify.
package node

import (
	"fmt"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

//Map that labels and values will be stored in
var nodes = map[string]*datamodel.Node{}

//Hard-coded string for log file warnings
var entityKind = "node"

//getNodeMetric takes data from prometheus and adds to nodes structure
func getNodeMetric(result model.Value, metric string) {

	if result == nil {
		return
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Check to see if we have a value for node field if not skip as won't be able to join to the system.
		nodeValue, ok := result.(model.Matrix)[i].Metric["node"]
		if !ok {
			continue
		}
		if _, ok := nodes[string(nodeValue)]; !ok {
			continue
		}
		switch metric {
		case "netSpeedBytes":
			//validates that the value of the entity is set and if not will default to 0
			if len(result.(model.Matrix)[i].Values) != 0 {
				nodes[string(nodeValue)].NetSpeedBytes = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
			}
		case "altWorkloadName":
			nodes[string(nodeValue)].AltWorkloadName = string(result.(model.Matrix)[i].Metric["instance"])
		default:
			if _, ok := nodes[string(nodeValue)].LabelMap[metric]; !ok {
				nodes[string(nodeValue)].LabelMap[metric] = map[string]string{}
			}
			for key, value := range result.(model.Matrix)[i].Metric {
				common.AddToLabelMap(string(key), string(value), nodes[string(nodeValue)].LabelMap[metric])
			}
		}

	}
}

//Metrics a global func for collecting node level metrics in prometheus
func Metrics(args *common.Parameters) {
	//Setup variables used in the code.
	var query string
	var result model.Value
	var err error

	//Query and store kubernetes node labels
	query = "max(kube_node_labels) by (instance, node)"
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.ErrorLogger.Println("metric=nodes query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=nodes query=" + query + " message=" + err.Error())
		return
	}
	var rsltIndex = result.(model.Matrix)
	for i := 0; i < rsltIndex.Len(); i++ {
		nodes[string(rsltIndex[i].Metric["node"])] =
			&datamodel.Node{LabelMap: map[string]map[string]string{}}
	}

	//query for node labels
	query = `kube_node_labels`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=nodeLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeLabels query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, query)
	}

	//query for node annotations
	query = `kube_node_annotations`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=nodeAnnotations query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeAnnotations query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, query)
	}

	//query for node info and store into labels structure
	query = `kube_node_info`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=nodeInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeInfo query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, query)
	}

	//query to get what roles may have been assigned to this node.
	query = `kube_node_role`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=nodeRole query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeRole query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, query)
	}

	//Gets the network speed in bytes as an attribute/config value for each node
	query = `label_replace(node_network_speed_bytes, "pod_ip", "$1", "instance", "(.*):.*")`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=networkSpeedBytes query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=networkSpeedBytes query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, "netSpeedBytes")
	}

	//Gets the conversion value for workload name for each node if needed.
	query = `max(max(label_replace(sum(node_cpu_seconds_total{mode!="idle"}) by (instance), "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip,instance) * on (pod_ip) group_left(node) kube_pod_info{pod=~".*node-exporter.*"}) by (node, instance)`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=altWorkloadName query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=altWorkloadName query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, "altWorkloadName")
	}

	var cluster = map[string]*datamodel.NodeCluster{}
	cluster["cluster"] = &datamodel.NodeCluster{Nodes: nodes, Name: *args.ClusterName}
	//Writes the config and attribute files
	common.WriteDiscovery(args, cluster, entityKind)

	//Queries the capacity fields of all nodes if we don't see any data then will try to use the older queries for capacity.
	query = `kube_node_status_capacity`
	result, err = common.MetricCollect(args, query, "discovery")
	if result.(model.Matrix).Len() == 0 {

		//Query and store prometheus total cpu cores
		query = `kube_node_status_capacity_cpu_cores`
		common.GetWorkload("node_capacity_cpu_cores", query, args, entityKind)

		//Query and store prometheus total memory in bytes.
		query = `kube_node_status_capacity_memory_bytes`
		common.GetWorkload("node_capacity_mem_bytes", query, args, entityKind)

		//Query and store prometheus total for pods per node.
		query = `kube_node_status_capacity_pods`
		common.GetWorkload("node_capacity_pods", query, args, entityKind)
	} else {
		//Query and store prometheus totals for capacity.
		query = `kube_node_status_capacity`
		common.GetWorkload("node_capacity", query, args, entityKind)
	}

	//Queries the allocatable fields of all nodes if we don't see data then will try to query the older metrics for allocations.
	query = `kube_node_status_allocatable`
	result, err = common.MetricCollect(args, query, "discovery")
	if result.(model.Matrix).Len() == 0 {

		//Query and store prometheus total cpu cores.
		query = `kube_node_status_allocatable_cpu_cores`
		common.GetWorkload("node_allocatable_cpu_cores", query, args, entityKind)

		//Query and store prometheus total memory in bytes
		query = `kube_node_status_allocatable_memory_bytes`
		common.GetWorkload("node_allocatable_mem_bytes", query, args, entityKind)

		//Query and store prometheus total for pods per node
		query = `kube_node_status_allocatable_pods`
		common.GetWorkload("node_allocatable_pods", query, args, entityKind)
	} else {
		//Query and store prometheus allocatable data
		query = `kube_node_status_allocatable`
		common.GetWorkload("node_allocatable", query, args, entityKind)
	}

	//Query and store prometheus total cpu uptime in seconds
	query = `node_cpu_seconds_total`
	common.GetWorkload("cpu_utilization", query, args, entityKind)

	//Query and store prometheus node memory total in bytes
	query = `node_memory_MemTotal_bytes`
	common.GetWorkload("memory_total_bytes", query, args, entityKind)

	//Query and store prometheus node memory total free in bytes
	query = `node_memory_MemFree_bytes`
	common.GetWorkload("memory_free_bytes", query, args, entityKind)

	//Query and store prometheus node memory cache in bytes
	query = `node_memory_Cached_bytes`
	common.GetWorkload("memory_cached_bytes", query, args, entityKind)

	//Query and store prometheus node memory buffers in bytes
	query = `node_memory_Buffers_bytes`
	common.GetWorkload("memory_buffers_bytes", query, args, entityKind)

	//Query and store prometheus node disk write in bytes
	query = `node_disk_written_bytes_total{device!~"dm-.*"}`
	common.GetWorkload("disk_write_bytes", query, args, entityKind)

	//Query and store prometheus node disk read in bytes
	query = `node_disk_read_bytes_total{device!~"dm-.*"}`
	common.GetWorkload("disk_read_bytes", query, args, entityKind)

	//Query and store prometheus total disk read uptime as a percentage
	query = `node_disk_read_time_seconds_total{device!~"dm-.*"}`
	common.GetWorkload("disk_read_ops", query, args, entityKind)

	//Query and store prometheus total disk write uptime as a percentage
	query = `node_disk_write_time_seconds_total{device!~"dm-.*"}`
	common.GetWorkload("disk_write_ops", query, args, entityKind)

	//Query and store prometheus total disk write uptime as a percentage
	query = `node_disk_io_time_seconds_total{device!~"dm-.*"}`
	common.GetWorkload("disk_total_ops", query, args, entityKind)

	//Query and store prometheus node recieved network data in bytes
	query = `node_network_receive_bytes_total{device!~"veth.*"}`
	common.GetWorkload("net_received_bytes", query, args, entityKind)

	//Query and store prometheus recieved network data in packets
	query = `node_network_receive_packets_total{device!~"veth.*"}`
	common.GetWorkload("net_received_packets", query, args, entityKind)

	//Query and store prometheus total transmitted network data in bytes
	query = `node_network_transmit_bytes_total{device!~"veth.*"}`
	common.GetWorkload("net_sent_bytes", query, args, entityKind)

	//Query and store prometheus total transmitted network data in packets
	query = `node_network_transmit_packets_total{device!~"veth.*"}`
	common.GetWorkload("net_sent_packets", query, args, entityKind)

}
