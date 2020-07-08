//Package nodegroup collects data related to containers and formats into csv files to send to Densify.
package nodegroup

import (
	"fmt"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

type nodeGroupStruct struct {
	nodes                                                                []string
	cpuLimit, cpuRequest, cpuCapacity, memLimit, memRequest, memCapacity int
}

var nodeGroups = map[string]*nodeGroupStruct{}

//Hard-coded string for log file warnings
var entityKind = "node_group"

//Gets node metrics from prometheus (and checks to see if they are valid)
func getNodeGroupMetric(result model.Value, nodeGroupLabel model.LabelName, metric string) {

	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		nodeGroup, ok := result.(model.Matrix)[i].Metric[nodeGroupLabel]
		if !ok {
			continue
		}
		if _, ok := nodeGroups[string(nodeGroup)]; !ok {
			continue
		}
		//validates that the value of the entity is set and if not will default to 0
		var value int
		if len(result.(model.Matrix)[i].Values) != 0 {
			value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
		}

		switch metric {
		case "cpuLimit":
			nodeGroups[string(nodeGroup)].cpuLimit = int(value)
		case "cpuRequest":
			nodeGroups[string(nodeGroup)].cpuRequest = int(value)
		case "cpuCapacity":
			nodeGroups[string(nodeGroup)].cpuCapacity = int(value)
		case "memLimit":
			nodeGroups[string(nodeGroup)].memLimit = int(value)
		case "memRequest":
			nodeGroups[string(nodeGroup)].memRequest = int(value)
		case "memCapacity":
			nodeGroups[string(nodeGroup)].memCapacity = int(value)
		}

	}
}

//writeNodeGroupConfig will create the config.csv file that is will be sent Densify by the Forwarder.
func writeNodeGroupConfig(args *common.Parameters) {

	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/node_group/config.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=node_group message=" + err.Error())
		fmt.Println("entity=node_group message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,node_group,HW Total CPUs,HW Total Physical CPUs,HW Cores Per CPU,HW Threads Per Core,HW Total Memory")

	for nodeGroupName, nodeGroup := range nodeGroups {
		fmt.Fprintf(configWrite, "%s,%s,", *args.ClusterName, nodeGroupName)

		if nodeGroup.cpuCapacity == -1 {
			fmt.Fprintf(configWrite, ",,1,1,")
		} else {
			fmt.Fprintf(configWrite, "%d,%d,1,1,", nodeGroup.cpuCapacity, nodeGroup.cpuCapacity)
		}

		if nodeGroup.memCapacity == -1 {
			fmt.Fprintf(configWrite, "\n")
		} else {
			fmt.Fprintf(configWrite, "%d\n", nodeGroup.memCapacity)
		}
	}
	configWrite.Close()
}

//writeNodeGroupAttributes will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeNodeGroupAttributes(args *common.Parameters) {

	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/node_group/attributes.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=node_group message=" + err.Error())
		fmt.Println("entity=node_group message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,node_group,Virtual Technology,Virtual Domain,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Current Size")

	for nodeGroupName, nodeGroup := range nodeGroups {
		//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
		fmt.Fprintf(attributeWrite, "%s,%s,NodeGroup,%s,", *args.ClusterName, nodeGroupName, *args.ClusterName)

		if nodeGroup.cpuLimit == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, "%d,", nodeGroup.cpuLimit)
		}

		if nodeGroup.cpuRequest == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, "%d,", nodeGroup.cpuRequest)
		}

		if nodeGroup.memLimit == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, "%d,", nodeGroup.memLimit)
		}

		if nodeGroup.memRequest == -1 {
			fmt.Fprintf(attributeWrite, "%d\n", len(nodeGroup.nodes))
		} else {
			fmt.Fprintf(attributeWrite, "%d,%d\n", nodeGroup.memRequest, len(nodeGroup.nodes))
		}
	}

	attributeWrite.Close()
}

//Metrics a global func for collecting node level metrics in prometheus
func Metrics(args *common.Parameters) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var query string
	var result model.Value

	//Start and end time + the prometheus address used for querying
	range5Min := common.TimeRange(args, historyInterval)

	// Node group set of queries
	var nodeGroupLabel model.LabelName

	query = `avg(kube_node_labels) by (label_cloud_google_com_gke_nodepool,label_eks_amazonaws_com_nodegroup, label_agentpool, label_pool_name)`
	result = common.MetricCollect(args, query, range5Min, "nodeGroupingLabelLookup", false)
	if result == nil {
		return
	}

	for i := range result.(model.Matrix) {
		for labelName := range result.(model.Matrix)[i].Metric {
			nodeGroupLabel = labelName
		}
	}

	if nodeGroupLabel == "" {
		return
	}

	query = `kube_node_labels{` + string(nodeGroupLabel) + `=~".+"}`
	result = common.MetricCollect(args, query, range5Min, "groupedNodes", false)
	if result == nil {
		return
	}
	for i := range result.(model.Matrix) {
		nodeGroup := string(result.(model.Matrix)[i].Metric[model.LabelName(nodeGroupLabel)])
		node := string(result.(model.Matrix)[i].Metric[`node`])
		if _, ok := nodeGroups[nodeGroup]; !ok {
			nodeGroups[nodeGroup] = &nodeGroupStruct{cpuLimit: -1, cpuRequest: -1, cpuCapacity: -1, memLimit: -1, memRequest: -1, memCapacity: -1}
		}
		nodeGroups[nodeGroup].nodes = append(nodeGroups[nodeGroup].nodes, node)
	}

	var nodeGroupSuffix = ` * on (node) group_right kube_node_labels{` + string(nodeGroupLabel) + `=~".+"}) by (` + string(nodeGroupLabel) + `)`

	query = `avg(sum(kube_pod_container_resource_limits_cpu_cores*1000 * on (namespace,pod,container) group_left kube_pod_container_status_running) by (node)` + nodeGroupSuffix
	result = common.MetricCollect(args, query, range5Min, "cpuLimit", false)
	if result != nil {
		getNodeGroupMetric(result, nodeGroupLabel, "cpuLimit")
	}

	query = `avg(sum(kube_pod_container_resource_requests_cpu_cores*1000 * on (namespace,pod,container) group_left kube_pod_container_status_running) by (node)` + nodeGroupSuffix
	result = common.MetricCollect(args, query, range5Min, "cpuRequest", false)
	if result != nil {
		getNodeGroupMetric(result, nodeGroupLabel, "cpuRequest")
	}

	query = `avg(sum(kube_pod_container_resource_limits_memory_bytes/1024/1024 * on (namespace,pod,container) group_left kube_pod_container_status_running) by (node)` + nodeGroupSuffix
	result = common.MetricCollect(args, query, range5Min, "memLimit", false)
	if result != nil {
		getNodeGroupMetric(result, nodeGroupLabel, "memLimit")
	}

	query = `avg(sum(kube_pod_container_resource_requests_memory_bytes/1024/1024 * on (namespace,pod,container) group_left kube_pod_container_status_running) by (node)` + nodeGroupSuffix
	result = common.MetricCollect(args, query, range5Min, "memRequest", false)
	if result != nil {
		getNodeGroupMetric(result, nodeGroupLabel, "memRequest")
	}

	query = `avg(kube_node_status_capacity_cpu_cores` + nodeGroupSuffix
	result = common.MetricCollect(args, query, range5Min, "cpuCapacity", false)
	if result != nil {
		getNodeGroupMetric(result, nodeGroupLabel, "cpuCapacity")
	}

	query = `avg(kube_node_status_capacity_memory_bytes/1024/1024` + nodeGroupSuffix
	result = common.MetricCollect(args, query, range5Min, "memCapacity", false)
	if result != nil {
		getNodeGroupMetric(result, nodeGroupLabel, "memCapacity")
	}

	writeNodeGroupAttributes(args)
	writeNodeGroupConfig(args)

	//Query and store prometheus CPU requests
	query = `avg(sum((kube_pod_container_resource_requests_cpu_cores) * on (namespace,pod,container) group_left kube_pod_container_status_running)  by (node)` + nodeGroupSuffix
	common.GetWorkload("cpu_requests", "CPU Reservation in Cores", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus CPU requests
	query = `avg(sum((kube_pod_container_resource_requests_cpu_cores) * on (namespace,pod,container) group_left kube_pod_container_status_running) by (node) / sum(kube_node_status_allocatable_cpu_cores) by (node)` + nodeGroupSuffix + ` * 100`
	common.GetWorkload("cpu_reservation_percent", "CPU Reservation Percent", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus Memory requests
	query = `avg(sum((kube_pod_container_resource_requests_memory_bytes/1024/1024) * on (namespace,pod,container) group_left kube_pod_container_status_running) by (node)` + nodeGroupSuffix
	common.GetWorkload("memory_requests", "Memory Reservation in MB", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus Memory requests
	query = `avg(sum((kube_pod_container_resource_requests_memory_bytes/1024/1024) * on (namespace,pod,container) group_left kube_pod_container_status_running) by (node) / sum(kube_node_status_allocatable_memory_bytes/1024/1024) by (node)` + nodeGroupSuffix + ` * 100`
	common.GetWorkload("memory_reservation_percent", "Memory Reservation Percent", query, nodeGroupLabel, args, entityKind)

	//Check to see which disk queries to use if instance is IP address that need to link to pod to get name or if instance = node name.
	query = `max(max(label_replace(sum(irate(node_cpu_seconds_total{mode!="idle"}[` + args.SampleRateString + `m])) by (instance) / on (instance) group_left count(node_cpu_seconds_total{mode="idle"}) by (instance) *100, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result = common.MetricCollect(args, query, range5Min, "testNodeWorkload", false)

	queryPrefix := `avg(label_replace(`
	queryPrefixSum := `avg(label_replace(sum(`
	querySuffix := `, "node", "$1", "instance", "(.*):*")` + nodeGroupSuffix
	querySuffixSum := `) by (instance), "node", "$1", "instance", "(.*):*")` + nodeGroupSuffix
	if result.(model.Matrix).Len() != 0 {
		queryPrefix = `avg(max(label_replace(`
		queryPrefixSum = `avg(sum(label_replace(`
		querySuffix = `, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}` + nodeGroupSuffix
		querySuffixSum = querySuffix
	}

	query = `sum(kube_node_labels{` + string(nodeGroupLabel) + `=~".+"}) by (` + string(nodeGroupLabel) + `)`
	common.GetWorkload("current_size", "Auto Scaling - In Service Instances", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus total cpu uptime in seconds
	query = queryPrefix + `sum(irate(node_cpu_seconds_total{mode!="idle"}[` + args.SampleRateString + `m])) by (instance) / on (instance) group_left count(node_cpu_seconds_total{mode="idle"}) by (instance) *100` + querySuffix
	common.GetWorkload("cpu_utilization", "CPU Utilization", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus node memory total in bytes
	query = queryPrefix + `node_memory_MemTotal_bytes - node_memory_MemFree_bytes` + querySuffix
	common.GetWorkload("memory_raw_bytes", "Raw Mem Utilization", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus node memory total free in bytes
	query = queryPrefix + `node_memory_MemTotal_bytes - (node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes)` + querySuffix
	common.GetWorkload("memory_actual_workload", "Actual Memory Utilization", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus node disk write in bytes
	query = queryPrefixSum + `irate(node_disk_written_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_write_bytes", "Raw Disk Write Utilization", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus node disk read in bytes
	query = queryPrefixSum + `irate(node_disk_read_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_read_bytes", "Raw Disk Read Utilization", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus total disk read uptime as a percentage
	query = queryPrefixSum + `irate(node_disk_read_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_read_ops", "Disk Read Operations", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus total disk write uptime as a percentage
	query = queryPrefixSum + `irate(node_disk_write_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_write_ops", "Disk Write Operations", query, nodeGroupLabel, args, entityKind)

	//Total disk values
	//Query and store prometheus node disk read in bytes
	query = queryPrefixSum + `irate(node_disk_read_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) + irate(node_disk_written_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_total_bytes", "Raw Disk Utilization", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus total disk read uptime as a percentage
	query = queryPrefixSum + `(irate(node_disk_read_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) + irate(node_disk_write_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("disk_total_ops", "Disk Operations", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus node recieved network data in bytes
	query = queryPrefixSum + `irate(node_network_receive_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_received_bytes", "Raw Net Received Utilization", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus recieved network data in packets
	query = queryPrefixSum + `irate(node_network_receive_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_received_packets", "Network Packets Received", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus total transmitted network data in bytes
	query = queryPrefixSum + `irate(node_network_transmit_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_sent_bytes", "Raw Net Sent Utilization", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus total transmitted network data in packets
	query = queryPrefixSum + `irate(node_network_transmit_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_sent_packets", "Network Packets Sent", query, nodeGroupLabel, args, entityKind)

	//Total values network
	//Query and store prometheus total network data in bytes
	query = queryPrefixSum + `irate(node_network_transmit_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m]) + irate(node_network_receive_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_total_bytes", "Raw Net Utilization", query, nodeGroupLabel, args, entityKind)

	//Query and store prometheus total network data in packets
	query = queryPrefixSum + `irate(node_network_transmit_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m]) + irate(node_network_receive_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	common.GetWorkload("net_total_packets", "Network Packets", query, nodeGroupLabel, args, entityKind)
}
