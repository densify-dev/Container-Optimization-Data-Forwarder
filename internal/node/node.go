//Package node collects data related to nodes and formats into json files to send to Densify.
package node

import (
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
	"net"
	"strings"
)

const (
	nodeKey           = prometheus.NodeEntityKind
	roleKey           = "role"
	deviceKey         = "device"
	instanceKey       = "instance"
	resourceKey       = "resource"
	unitKey           = "unit"
	underscore        = "_"
	integer           = "integer"
	capacityMetric    = `kube_node_status_capacity`
	allocatableMetric = `kube_node_status_allocatable`
	cpuCores          = "cpu_cores"
	memBytes          = "memory_bytes"
	pods              = "pods"
)

// Map of labels and values
var nodes = make(map[string]*datamodel.Node)
var nodesByAltName = make(map[string]*datamodel.Node)
var altNameMap = make(map[string]string)

var nodeNameKeys = map[string]string{"netSpeedBytes": instanceKey}

var podIpFilter = []string{datamodel.PodIpKey}
var deviceKeys = []string{deviceKey}
var resourceKeys = []string{resourceKey, unitKey}
var resourceSubQueries = []string{cpuCores, memBytes, pods}
var conditionKeys = []string{datamodel.ConditionKey}

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
			//		case "kube_node_status_condition":
			// TODO deal with different conditions!
		case "netSpeedBytes":
			if no, ok = getNode(nodeName); !ok {
				continue
			}
			device, err := datamodel.GetActualKey(mat[i].Metric, deviceKeys, true)
			if err != nil {
				continue
			}
			deviceLabels := datamodel.EnsureLabels(no.NetSpeedBytesMap, device)
			_ = deviceLabels.AppendSampleStreamWithValue(mat[i], device, nil)
		case "altWorkloadName":
			if no, ok = nodes[nodeName]; !ok {
				continue
			}
			podIp := string(mat[i].Metric[datamodel.PodIpKey])
			// now insert the node to nodesByAltName
			if _, f := nodesByAltName[podIp]; !f {
				nodesByAltName[podIp] = no
				altNameMap[podIp] = nodeName
			}
			// no need to update anything, this is just for mapping by altName
		case "nodeStatusCondition":
			if no, ok = getNode(nodeName); !ok {
				continue
			}
			var cond string
			var err error
			if cond, err = datamodel.GetActualKey(mat[i].Metric, conditionKeys, true); err != nil {
				continue
			}
			var condition *datamodel.Condition
			var f bool
			if condition, f = no.RawConditions[cond]; !f {
				condition = &datamodel.Condition{}
				no.RawConditions[cond] = condition
			}
			if err = condition.Append(mat[i]); err != nil {
				continue
			}
			if condition.IsComplete() {
				var ss *model.SampleStream
				if ss, err = condition.Consolidate(); err != nil {
					continue
				}
				conditionLabels := datamodel.EnsureLabels(no.Conditions, cond)
				_ = conditionLabels.AppendSampleStreamWithValue(ss, cond, nil)
			}
		default:
			if no, ok = nodes[nodeName]; !ok {
				continue
			}
			var capacityOrAllocatable bool
			if capacityOrAllocatable = getCapacityAllocatableMetric(no.Capacity, mat[i], metric, capacityMetric); !capacityOrAllocatable {
				capacityOrAllocatable = getCapacityAllocatableMetric(no.Allocatable, mat[i], metric, allocatableMetric)
			}
			if !capacityOrAllocatable {
				// labels
				labels := datamodel.EnsureLabels(no.LabelMap, metric)
				_ = labels.AppendSampleStream(mat[i])
			}
		}
	}
}

func getCapacityAllocatableMetrics(queryGroup string, args *prometheus.Parameters) {
	query := queryGroup
	result, err := prometheus.MetricCollect(args, query)
	var trySubQueries bool
	if err != nil {
		args.ErrorLogger.Println("metric=nodes query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=nodes query=" + query + " message=" + err.Error())
		trySubQueries = true
	} else if result == nil || result.(model.Matrix).Len() == 0 {
		trySubQueries = true
	}
	if trySubQueries {
		for _, subQuery := range resourceSubQueries {
			query = strings.Join([]string{queryGroup, subQuery}, underscore)
			if result, err = prometheus.MetricCollect(args, query); err == nil {
				getNodeMetric(result, query)
			} else {
				args.ErrorLogger.Println("metric=nodes query=" + query + " message=" + err.Error())
				fmt.Println("[ERROR] metric=nodes query=" + query + " message=" + err.Error())
			}
		}
	} else {
		getNodeMetric(result, query)
	}
}

func getCapacityAllocatableMetric(lm datamodel.LabelMap, ss *model.SampleStream, metric, metricGroup string) bool {
	var ok bool
	if sc, f := getSuffixComponents(metric, metricGroup); f {
		var keys []string
		var replaceKeysByValues bool
		if n := len(sc); n == 0 {
			// new API, kube_node_status_capacity or kube_node_status_allocatable
			keys = resourceKeys
			replaceKeysByValues = true
		} else {
			keys = sc
			if n == 1 {
				// pods
				keys = append(keys, integer)
			} else {
				sc[1] = strings.TrimSuffix(sc[1], "s")
			}
		}
		var ak string
		var err error
		if ak, err = datamodel.GetActualKey(ss.Metric, keys[:1], replaceKeysByValues); err == nil {
			l := datamodel.EnsureLabels(lm, ak)
			if ak, err = datamodel.GetActualKey(ss.Metric, keys[1:], replaceKeysByValues); err == nil {
				err = l.AppendSampleStreamWithValue(ss, ak, nil)
			}
		}
		ok = err == nil
	}
	return ok
}

func getSuffixComponents(metric, metricGroup string) (s []string, f bool) {
	suffix := strings.TrimPrefix(metric, metricGroup)
	n := len(suffix)
	switch n {
	case 0:
		// exact match
		f = true
	case len(metric):
		// no match, f is false already
	default:
		// metricGroup is a prefix of metric
		sc := strings.SplitN(suffix, underscore, 2)
		if k := len(sc); k > 0 {
			f = true
			if sc[0] == "" {
				if k > 1 {
					s = sc[1:]
				}
			} else {
				s = sc
			}
		}
	}
	return
}

//Metrics a global func for collecting node level metrics in prometheus
func Metrics(args *prometheus.Parameters) {
	//Setup variables used in the code.
	var query string
	var result model.Value
	var err error

	//Query and store kubernetes node labels
	query = `kube_node_labels`
	result, err = prometheus.MetricCollect(args, query)
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
				nodes[nodeName] = datamodel.NewNode()
			}
		}
	}
	getNodeMetric(result, query)

	//query for node annotations
	query = `kube_node_annotations`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=nodeAnnotations query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeAnnotations query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, query)
	}

	//query for node info and store into labels structure
	query = `kube_node_info`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=nodeInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeInfo query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, query)
	}

	//query to get what roles may have been assigned to this node.
	query = `kube_node_role`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=nodeRole query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeRole query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, query)
	}

	// Gets the alternative node name (pod_ip of node exporter)
	// This needs to be done before node_network_speed_bytes
	query = `kube_pod_info{pod=~".*node-exporter.*"}`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=altWorkloadName query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=altWorkloadName query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, "altWorkloadName")
	}

	//Gets the network speed in bytes as an attribute/config value for each node
	query = `node_network_speed_bytes{device!~"veth.*"}`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=networkSpeedBytes query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=networkSpeedBytes query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, "netSpeedBytes")
	}

	getCapacityAllocatableMetrics(capacityMetric, args)
	getCapacityAllocatableMetrics(allocatableMetric, args)

	// Query and store the node status condition
	query = `kube_node_status_condition`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=kube_node_status_condition query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=kube_node_status_condition query=" + query + " message=" + err.Error())
	} else {
		getNodeMetric(result, "nodeStatusCondition")
	}

	if disc, err := args.ToDiscovery(prometheus.NodeEntityKind); err == nil {
		discovery := &datamodel.NodeDiscovery{Discovery: disc, Nodes: nodes}
		prometheus.WriteDiscovery(args, discovery, prometheus.NodeEntityKind)
	}

	//Query and store prometheus total cpu uptime in seconds
	query = `node_cpu_seconds_total`
	getNodeExporterWorkload("cpu_utilization", query, args)

	//Query and store prometheus node memory total in bytes
	query = `node_memory_MemTotal_bytes`
	getNodeExporterWorkload("memory_total_bytes", query, args)

	//Query and store prometheus node memory total free in bytes
	query = `node_memory_MemFree_bytes`
	getNodeExporterWorkload("memory_free_bytes", query, args)

	//Query and store prometheus node memory cache in bytes
	query = `node_memory_Cached_bytes`
	getNodeExporterWorkload("memory_cached_bytes", query, args)

	//Query and store prometheus node memory buffers in bytes
	query = `node_memory_Buffers_bytes`
	getNodeExporterWorkload("memory_buffers_bytes", query, args)

	//Query and store prometheus node disk write in bytes
	query = `node_disk_written_bytes_total{device!~"dm-.*"}`
	getNodeExporterWorkload("disk_write_bytes", query, args)

	//Query and store prometheus node disk read in bytes
	query = `node_disk_read_bytes_total{device!~"dm-.*"}`
	getNodeExporterWorkload("disk_read_bytes", query, args)

	//Query and store prometheus total disk read uptime as a percentage
	query = `node_disk_read_time_seconds_total{device!~"dm-.*"}`
	getNodeExporterWorkload("disk_read_ops", query, args)

	//Query and store prometheus total disk write uptime as a percentage
	query = `node_disk_write_time_seconds_total{device!~"dm-.*"}`
	getNodeExporterWorkload("disk_write_ops", query, args)

	//Query and store prometheus total disk write uptime as a percentage
	query = `node_disk_io_time_seconds_total{device!~"dm-.*"}`
	getNodeExporterWorkload("disk_total_ops", query, args)

	//Query and store prometheus node received network data in bytes
	query = `node_network_receive_bytes_total{device!~"veth.*"}`
	getNodeExporterWorkload("net_received_bytes", query, args)

	//Query and store prometheus received network data in packets
	query = `node_network_receive_packets_total{device!~"veth.*"}`
	getNodeExporterWorkload("net_received_packets", query, args)

	//Query and store prometheus total transmitted network data in bytes
	query = `node_network_transmit_bytes_total{device!~"veth.*"}`
	getNodeExporterWorkload("net_sent_bytes", query, args)

	//Query and store prometheus total transmitted network data in packets
	query = `node_network_transmit_packets_total{device!~"veth.*"}`
	getNodeExporterWorkload("net_sent_packets", query, args)

}

var nera = &prometheus.RelabelArgs{
	Key: instanceKey,
	Map: altNameMap,
	VCF: hostName,
}

func getNodeExporterWorkload(filename, query string, args *prometheus.Parameters) {
	prometheus.GetFilteredRelabeledWorkload(filename, query, args, prometheus.NodeEntityKind, nil, nera)
}

func hostName(name string) (string, bool) {
	// strip off port, if part of addr
	if host, _, err := net.SplitHostPort(name); err == nil {
		return host, true
	} else {
		return name, false
	}
}

func getNode(name string) (*datamodel.Node, bool) {
	var no *datamodel.Node
	var ok bool
	if nodeName, isAlt := hostName(name); isAlt {
		no, ok = nodesByAltName[nodeName]
	} else {
		no, ok = nodes[nodeName]
	}
	return no, ok
}
