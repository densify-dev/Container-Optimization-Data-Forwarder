//Package node collects data related to nodes and formats into json files to send to Densify.
package node

import (
	"fmt"
	"net"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

// Map of labels and values
var nodes = make(map[string]*datamodel.Node)
var nodesByAltName = make(map[string]*datamodel.Node)

var nodeNameKeys = map[string]string{"netSpeedBytes": "instance"}

const (
	nodeKey   = common.NodeEntityKind
	podIpKey  = "pod_ip"
	roleKey   = "role"
	deviceKey = "device"
)

var podIpFilter = []string{podIpKey}

//getNodeMetric takes data from prometheus and adds to nodes structure
func getNodeMetric(result model.Value, metric string) {
	mat, ok := result.(model.Matrix)
	if !ok {
		return
	}
	nodeNameKey, ok := nodeNameKeys[metric]
	if !ok {
		nodeNameKey = nodeKey
	}
	// Loop through the different entities in the results.
	n := mat.Len()
	for i := 0; i < n; i++ {
		nodeNameValue, ok := mat[i].Metric[model.LabelName(nodeNameKey)]
		if !ok {
			continue
		}
		nodeName := string(nodeNameValue)
		var no *datamodel.Node
		switch metric {
		case "kube_node_role":
			if no, ok = nodes[nodeName]; !ok {
				continue
			}
			role := string(mat[i].Metric[roleKey])
			roleLabels := datamodel.EnsureLabels(no.Roles, role)
			_ = roleLabels.AppendSampleStream(mat[i])
		case "netSpeedBytes":
			// strip off port, if part of addr
			if host, _, err := net.SplitHostPort(nodeName); err == nil {
				nodeName = host
			}
			// first try in nodes
			if no, ok = nodes[nodeName]; !ok {
				// then in nodes by alt-name
				if no, ok = nodesByAltName[nodeName]; !ok {
					continue
				}
			}
			device := string(mat[i].Metric[deviceKey])
			deviceLabels := datamodel.EnsureLabels(no.NetSpeedBytesMap, device)
			_ = deviceLabels.AppendSampleStreamWithValue(mat[i], deviceKey, nil)
		case "altWorkloadName":
			if no, ok = nodes[nodeName]; !ok {
				continue
			}
			podIp := string(mat[i].Metric[podIpKey])
			// now insert the node to nodesByAltName
			if _, f := nodesByAltName[podIp]; !f {
				nodesByAltName[podIp] = no
			}
			_ = no.AltWorkloadName.AppendSampleStreamWithFilter(mat[i], podIpFilter)
		default:
			if no, ok = nodes[nodeName]; !ok {
				continue
			}
			labels := datamodel.EnsureLabels(no.LabelMap, metric)
			_ = labels.AppendSampleStream(mat[i])
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
	query = `kube_node_labels`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.ErrorLogger.Println("metric=nodes query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=nodes query=" + query + " message=" + err.Error())
		return
	}
	mat := result.(model.Matrix)
	n := mat.Len()
	for i := 0; i < n; i++ {
		if nodeName := string(mat[i].Metric[nodeKey]); nodeName != "" {
			if _, ok := nodes[nodeName]; !ok {
				nodes[nodeName] = &datamodel.Node{
					LabelMap:         make(datamodel.LabelMap),
					Roles:            make(datamodel.LabelMap),
					NetSpeedBytesMap: make(datamodel.LabelMap),
					AltWorkloadName:  &datamodel.Labels{},
				}
			}
		}
	}
	getNodeMetric(result, query)

	//query for node annotations
	query = `kube_node_annotations`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=nodeAnnotations query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeAnnotations query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, query)
	}

	//query for node info and store into labels structure
	query = `kube_node_info`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=nodeInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeInfo query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, query)
	}

	//query to get what roles may have been assigned to this node.
	query = `kube_node_role`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=nodeRole query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeRole query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, query)
	}

	// Gets the alternative node name (pod_ip of node exporter)
	// This needs to be done before node_network_speed_bytes
	query = `kube_pod_info{pod=~".*node-exporter.*"}`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=altWorkloadName query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=altWorkloadName query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, "altWorkloadName")
	}

	//Gets the network speed in bytes as an attribute/config value for each node
	query = `node_network_speed_bytes{device!~"veth.*"}`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=networkSpeedBytes query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=networkSpeedBytes query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, "netSpeedBytes")
	}

	if disc, err := args.ToDiscovery(common.NodeEntityKind); err == nil {
		discovery := &datamodel.NodeDiscovery{Discovery: disc, Nodes: nodes}
		common.WriteDiscovery(args, discovery, common.NodeEntityKind)
	}

	//Queries the capacity fields of all nodes if we don't see any data then will try to use the older queries for capacity.
	query = `kube_node_status_capacity`
	result, err = common.MetricCollect(args, query)
	if result.(model.Matrix).Len() == 0 {

		//Query and store prometheus total cpu cores
		query = `kube_node_status_capacity_cpu_cores`
		common.GetWorkload("node_capacity_cpu_cores", query, args, common.NodeEntityKind)

		//Query and store prometheus total memory in bytes.
		query = `kube_node_status_capacity_memory_bytes`
		common.GetWorkload("node_capacity_mem_bytes", query, args, common.NodeEntityKind)

		//Query and store prometheus total for pods per node.
		query = `kube_node_status_capacity_pods`
		common.GetWorkload("node_capacity_pods", query, args, common.NodeEntityKind)
	} else {
		//Query and store prometheus totals for capacity.
		query = `kube_node_status_capacity`
		common.GetWorkload("node_capacity", query, args, common.NodeEntityKind)
	}

	//Queries the allocatable fields of all nodes if we don't see data then will try to query the older metrics for allocations.
	query = `kube_node_status_allocatable`
	result, err = common.MetricCollect(args, query)
	if result.(model.Matrix).Len() == 0 {

		//Query and store prometheus total cpu cores.
		query = `kube_node_status_allocatable_cpu_cores`
		common.GetWorkload("node_allocatable_cpu_cores", query, args, common.NodeEntityKind)

		//Query and store prometheus total memory in bytes
		query = `kube_node_status_allocatable_memory_bytes`
		common.GetWorkload("node_allocatable_mem_bytes", query, args, common.NodeEntityKind)

		//Query and store prometheus total for pods per node
		query = `kube_node_status_allocatable_pods`
		common.GetWorkload("node_allocatable_pods", query, args, common.NodeEntityKind)
	} else {
		//Query and store prometheus allocatable data
		query = `kube_node_status_allocatable`
		common.GetWorkload("node_allocatable", query, args, common.NodeEntityKind)
	}

	//Query and store prometheus total cpu uptime in seconds
	query = `node_cpu_seconds_total`
	common.GetWorkload("cpu_utilization", query, args, common.NodeEntityKind)

	//Query and store prometheus node memory total in bytes
	query = `node_memory_MemTotal_bytes`
	common.GetWorkload("memory_total_bytes", query, args, common.NodeEntityKind)

	//Query and store prometheus node memory total free in bytes
	query = `node_memory_MemFree_bytes`
	common.GetWorkload("memory_free_bytes", query, args, common.NodeEntityKind)

	//Query and store prometheus node memory cache in bytes
	query = `node_memory_Cached_bytes`
	common.GetWorkload("memory_cached_bytes", query, args, common.NodeEntityKind)

	//Query and store prometheus node memory buffers in bytes
	query = `node_memory_Buffers_bytes`
	common.GetWorkload("memory_buffers_bytes", query, args, common.NodeEntityKind)

	//Query and store prometheus node disk write in bytes
	query = `node_disk_written_bytes_total{device!~"dm-.*"}`
	common.GetWorkload("disk_write_bytes", query, args, common.NodeEntityKind)

	//Query and store prometheus node disk read in bytes
	query = `node_disk_read_bytes_total{device!~"dm-.*"}`
	common.GetWorkload("disk_read_bytes", query, args, common.NodeEntityKind)

	//Query and store prometheus total disk read uptime as a percentage
	query = `node_disk_read_time_seconds_total{device!~"dm-.*"}`
	common.GetWorkload("disk_read_ops", query, args, common.NodeEntityKind)

	//Query and store prometheus total disk write uptime as a percentage
	query = `node_disk_write_time_seconds_total{device!~"dm-.*"}`
	common.GetWorkload("disk_write_ops", query, args, common.NodeEntityKind)

	//Query and store prometheus total disk write uptime as a percentage
	query = `node_disk_io_time_seconds_total{device!~"dm-.*"}`
	common.GetWorkload("disk_total_ops", query, args, common.NodeEntityKind)

	//Query and store prometheus node recieved network data in bytes
	query = `node_network_receive_bytes_total{device!~"veth.*"}`
	common.GetWorkload("net_received_bytes", query, args, common.NodeEntityKind)

	//Query and store prometheus recieved network data in packets
	query = `node_network_receive_packets_total{device!~"veth.*"}`
	common.GetWorkload("net_received_packets", query, args, common.NodeEntityKind)

	//Query and store prometheus total transmitted network data in bytes
	query = `node_network_transmit_bytes_total{device!~"veth.*"}`
	common.GetWorkload("net_sent_bytes", query, args, common.NodeEntityKind)

	//Query and store prometheus total transmitted network data in packets
	query = `node_network_transmit_packets_total{device!~"veth.*"}`
	common.GetWorkload("net_sent_packets", query, args, common.NodeEntityKind)

}
