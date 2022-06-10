//Package nodegroup collects data related to containers and formats into csv files to send to Densify.
package nodegroup

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

type nodeGroupStruct struct {
	nodes                                                                             string
	cpuLimit, cpuRequest, cpuCapacity, memLimit, memRequest, memCapacity, currentSize int
	labelMap                                                                          map[string]string
}

var nodeGroups = map[string]*nodeGroupStruct{}

//Hard-coded string for log file warnings
var entityKind = "node_group"

//getNodeMetricString is used to parse the label based results from Prometheus related to Container Entities and store them in the systems data structure.
func getNodeMetricString(result model.Value, nodeGroup model.LabelName) {
	//Validate there is data in the results.
	if result == nil {
		return
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		nodeGroupValue, ok := result.(model.Matrix)[i].Metric[nodeGroup]
		if !ok {
			continue
		}
		if _, ok := nodeGroups[string(nodeGroupValue)]; !ok {
			continue
		}
		for key, value := range result.(model.Matrix)[i].Metric {
			common.AddToLabelMap(string(key), string(value), nodeGroups[string(nodeGroupValue)].labelMap)
		}
	}
}

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
		case "capacity":
			switch result.(model.Matrix)[i].Metric["resource"] {
			case "cpu":
				nodeGroups[string(nodeGroup)].cpuCapacity = int(value)
			case "memory":
				nodeGroups[string(nodeGroup)].memCapacity = int(value)
			}
		}
	}
}

//writeNodeGroupConfig will create the config.csv file that is will be sent to Densify by the Forwarder.
func writeConfig(args *common.Parameters) {

	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/node_group/config.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "AuditTime,ClusterName,NodeGroupName,HwTotalCpus,HwTotalPhysicalCpus,HwCoresPerCpu,HwThreadsPerCore,HwTotalMemory,HwModel,OsName")

	for nodeGroupName, nodeGroup := range nodeGroups {
		var os, instance string
		if _, ok := nodeGroup.labelMap["label_kubernetes_io_os"]; ok {
			os = "label_kubernetes_io_os"
		} else {
			os = "label_beta_kubernetes_io_os"
		}

		if value, ok := nodeGroup.labelMap["label_node_kubernetes_io_instance_type"]; ok {
			instance = value
		} else if value, ok := nodeGroup.labelMap["label_beta_kubernetes_io_instance_type"]; ok {
			instance = value
		} else {
			instance = ""
		}

		fmt.Fprintf(configWrite, "%s,%s,%s,", common.Format(args.CurrentTime), *args.ClusterName, nodeGroupName)

		if nodeGroup.cpuCapacity == -1 {
			fmt.Fprintf(configWrite, ",,1,1,")
		} else {
			fmt.Fprintf(configWrite, "%d,%d,1,1,", nodeGroup.cpuCapacity, nodeGroup.cpuCapacity)
		}
		if nodeGroup.memCapacity == -1 {
			fmt.Fprintf(configWrite, ",%s,%s\n", instance, nodeGroup.labelMap[os])
		} else {
			fmt.Fprintf(configWrite, "%d,%s,%s\n", nodeGroup.memCapacity, instance, nodeGroup.labelMap[os])
		}
	}
	configWrite.Close()
}

//writeNodeGroupAttributes will create the attributes.csv file that is will be sent to Densify by the Forwarder.
func writeAttributes(args *common.Parameters) {

	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/node_group/attributes.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "ClusterName,NodeGroupName,VirtualTechnology,VirtualDomain,CpuLimit,CpuRequest,MemoryLimit,MemoryRequest,CurrentSize,CurrentNodes,NodeLabels")
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
			fmt.Fprintf(attributeWrite, "%d,%s,", nodeGroup.currentSize, nodeGroup.nodes[:len(nodeGroup.nodes)-1])
		} else {
			fmt.Fprintf(attributeWrite, "%d,%d,%s,", nodeGroup.memRequest, nodeGroup.currentSize, nodeGroup.nodes[:len(nodeGroup.nodes)-1])
		}
		for key, value := range nodeGroup.labelMap {
			if len(key) >= 250 {
				continue
			}
			value = strings.Replace(value, ",", " ", -1)
			if len(value)+3+len(key) < 256 {
				fmt.Fprintf(attributeWrite, key+" : "+value+"|")
			} else {
				templength := 256 - 3 - len(key)
				fmt.Fprintf(attributeWrite, key+" : "+value[:templength]+"|")
			}
		}
		fmt.Fprintf(attributeWrite, "\n")
	}

	attributeWrite.Close()
}

//checkNodeGroups checks to see if the node group label in the results is already in the nodeGroupsLabels array or not.
func checkNodeGroups(nodeGroupLabels []model.LabelName, labelName model.LabelName) bool {
	for _, label := range nodeGroupLabels {
		if label == labelName {
			return true
		}
	}
	return false
}

//getWorkload used to query for the workload data and then calls write workload
func getWorkload(fileName, metricName, query string, nodeGroupLabels []model.LabelName, args *common.Parameters, entityKind string) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/" + entityKind + "/" + fileName + ".csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("entity=" + entityKind + " message=" + err.Error())
		return
	}
	if csvHeaderFormat, f := common.GetCsvHeaderFormat(entityKind); f {
		fmt.Fprintf(workloadWrite, csvHeaderFormat, metricName)
	} else {
		msg := " message=no CSV header format found"
		args.ErrorLogger.Println("entity=" + entityKind + msg)
		fmt.Println("entity=" + entityKind + msg)
		return
	}

	for _, metricField := range nodeGroupLabels {

		query2 := strings.ReplaceAll(query, "stringToBeReplaced", string(metricField))

		//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
		//This is done as the farther you go back in time the slower prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
		//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
		for historyInterval = 0; int(historyInterval) < *args.History; historyInterval++ {
			range5Min := common.TimeRange(args, historyInterval)

			result, err = common.MetricCollect(args, query2, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=" + metricName + " query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=" + metricName + " query=" + query + " message=" + err.Error())
			} else {
				var field []model.LabelName
				field = append(field, metricField)
				common.WriteWorkload(workloadWrite, result, field, args, entityKind)
			}
		}
	}
	//Close the workload files.
	workloadWrite.Close()
}

//Metrics a global func for collecting node level metrics in prometheus
func Metrics(args *common.Parameters) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var query string
	var result model.Value
	var err error

	//Start and end time + the prometheus address used for querying
	range5Min := common.TimeRange(args, historyInterval)

	// Node group set of queries
	var nodeGroupLabels []model.LabelName

	query = `avg(kube_node_labels) by (` + args.NodeGroupList + `)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=nodeGroup query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nodeGroup query=" + query + " message=" + err.Error())
		return
	}

	for i := range result.(model.Matrix) {
		for labelName := range result.(model.Matrix)[i].Metric {
			labelFound := checkNodeGroups(nodeGroupLabels, labelName)
			if !labelFound {
				nodeGroupLabels = append(nodeGroupLabels, labelName)
			}
		}
	}

	if len(nodeGroupLabels) == 0 {
		return
	}
	var nodeGroupSuffix string
	var requestsLabel string

	for ng := range nodeGroupLabels {
		query = `kube_node_labels{` + string(nodeGroupLabels[ng]) + `=~".+"}`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.ErrorLogger.Println("metric=groupedNodes query=" + query + " message=" + err.Error())
			fmt.Println("[ERROR] metric=groupedNodes query=" + query + " message=" + err.Error())
			continue
		}
		for i := range result.(model.Matrix) {
			nodeGroup := string(result.(model.Matrix)[i].Metric[nodeGroupLabels[ng]])
			node := string(result.(model.Matrix)[i].Metric[`node`])
			if _, ok := nodeGroups[nodeGroup]; !ok {
				nodeGroups[nodeGroup] = &nodeGroupStruct{cpuLimit: -1, cpuRequest: -1, cpuCapacity: -1, memLimit: -1, memRequest: -1, memCapacity: -1, labelMap: map[string]string{}}
			}
			nodeGroups[nodeGroup].nodes = nodeGroups[nodeGroup].nodes + node + "|"
			nodeGroups[nodeGroup].currentSize++
		}

		getNodeMetricString(result, nodeGroupLabels[ng])

		nodeGroupSuffix = ` * on (node) group_left (` + string(nodeGroupLabels[ng]) + `) kube_node_labels{` + string(nodeGroupLabels[ng]) + `=~".+"}) by (` + string(nodeGroupLabels[ng]) + `)`

		query = `sum(kube_pod_container_resource_limits) by (node, resource)`
		result, err = common.MetricCollect(args, query, range5Min)
		if result.(model.Matrix).Len() == 0 {
			query = `avg(sum(kube_pod_container_resource_limits_cpu_cores*1000) by (node)` + nodeGroupSuffix
			result, err = common.MetricCollect(args, query, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=cpuLimit query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=cpuLimit query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "cpuLimit")
			}

			query = `avg(sum(kube_pod_container_resource_limits_memory_bytes/1024/1024) by (node)` + nodeGroupSuffix
			result, err = common.MetricCollect(args, query, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=memLimit query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=memLimit query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "memLimit")
			}

		} else {
			query = `avg(sum(kube_pod_container_resource_limits{resource="cpu"}*1000) by (node)` + nodeGroupSuffix
			result, err = common.MetricCollect(args, query, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=cpuLimit query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=cpuLimit query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "cpuLimit")
			}

			query = `avg(sum(kube_pod_container_resource_limits{resource="memory"}/1024/1024) by (node)` + nodeGroupSuffix
			result, err = common.MetricCollect(args, query, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=memLimit query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=memLimit query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "memLimit")
			}

		}

		query = `sum(kube_pod_container_resource_requests) by (node, resource)`
		result, err = common.MetricCollect(args, query, range5Min)
		if result.(model.Matrix).Len() == 0 {
			query = `avg(sum(kube_pod_container_resource_requests_cpu_cores*1000) by (node)` + nodeGroupSuffix
			result, err = common.MetricCollect(args, query, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=cpuRequest query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=cpuRequest query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "cpuRequest")
			}

			query = `avg(sum(kube_pod_container_resource_requests_memory_bytes/1024/1024) by (node)` + nodeGroupSuffix
			result, err = common.MetricCollect(args, query, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=memRequest query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=memRequest query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "memRequest")
			}
		} else {
			query = `avg(sum(kube_pod_container_resource_requests{resource="cpu"}*1000) by (node)` + nodeGroupSuffix
			result, err = common.MetricCollect(args, query, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=cpuRequest query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=cpuRequest query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "cpuRequest")
			}

			query = `avg(sum(kube_pod_container_resource_requests{resource="memory"}/1024/1024) by (node)` + nodeGroupSuffix
			result, err = common.MetricCollect(args, query, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=memRequest query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=memRequest query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "memRequest")
			}
			requestsLabel = "unified"
		}

		query = `avg(kube_node_status_capacity * on (node) group_left (` + string(nodeGroupLabels[ng]) + `) kube_node_labels{` + string(nodeGroupLabels[ng]) + `=~".+"}) by (` + string(nodeGroupLabels[ng]) + `,resource)`
		result, err = common.MetricCollect(args, query, range5Min)

		if result.(model.Matrix).Len() == 0 {
			query = `avg(kube_node_status_capacity_cpu_cores` + nodeGroupSuffix
			result, err = common.MetricCollect(args, query, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=cpuCapacity query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=cpuCapacity query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "cpuCapacity")
			}

			query = `avg(kube_node_status_capacity_memory_bytes/1024/1024` + nodeGroupSuffix
			result, err = common.MetricCollect(args, query, range5Min)
			if err != nil {
				args.WarnLogger.Println("metric=memCapacity query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=memCapacity query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "memCapacity")
			}
		} else {
			if err != nil {
				args.WarnLogger.Println("metric=statusCapacity query=" + query + " message=" + err.Error())
				fmt.Println("[WARNING] metric=statusCapacity query=" + query + " message=" + err.Error())
			} else {
				getNodeGroupMetric(result, nodeGroupLabels[ng], "capacity")
			}
		}
	}
	writeAttributes(args)
	writeConfig(args)

	//reset the nodeGroupSuffix with value that can be searched for and replaced easily as go through each workload.
	nodeGroupSuffix = ` * on (node) group_right kube_node_labels{stringToBeReplaced=~".+"}) by (stringToBeReplaced)`

	if requestsLabel == "unified" {
		//Query and store prometheus CPU requests
		query = `avg(sum(kube_pod_container_resource_requests{resource="cpu"})  by (node)` + nodeGroupSuffix
		getWorkload("cpu_requests", "CpuRequests", query, nodeGroupLabels, args, entityKind)

		//Query and store prometheus CPU requests
		query = `avg(sum(kube_pod_container_resource_requests{resource="cpu"}) by (node) / sum(kube_node_status_capacity{resource="cpu"}) by (node)` + nodeGroupSuffix + ` * 100`
		getWorkload("cpu_reservation_percent", "CpuReservationPercent", query, nodeGroupLabels, args, entityKind)

		//Query and store prometheus Memory requests
		query = `avg(sum(kube_pod_container_resource_requests{resource="memory"}/1024/1024) by (node)` + nodeGroupSuffix
		getWorkload("memory_requests", "MemoryRequests", query, nodeGroupLabels, args, entityKind)

		//Query and store prometheus Memory requests
		query = `avg(sum(kube_pod_container_resource_requests{resource="memory"}/1024/1024) by (node) / sum(kube_node_status_capacity{resource="memory"}/1024/1024) by (node)` + nodeGroupSuffix + ` * 100`
		getWorkload("memory_reservation_percent", "MemoryReservationPercent", query, nodeGroupLabels, args, entityKind)
	} else {
		//Query and store prometheus CPU requests
		query = `avg(sum(kube_pod_container_resource_requests_cpu_cores)  by (node)` + nodeGroupSuffix
		getWorkload("cpu_requests", "CpuRequests", query, nodeGroupLabels, args, entityKind)

		//Query and store prometheus CPU requests
		query = `avg(sum(kube_pod_container_resource_requests_cpu_cores) by (node) / sum(kube_node_status_capacity_cpu_cores) by (node)` + nodeGroupSuffix + ` * 100`
		getWorkload("cpu_reservation_percent", "CpuReservationPercent", query, nodeGroupLabels, args, entityKind)

		//Query and store prometheus Memory requests
		query = `avg(sum(kube_pod_container_resource_requests_memory_bytes/1024/1024) by (node)` + nodeGroupSuffix
		getWorkload("memory_requests", "MemoryRequests", query, nodeGroupLabels, args, entityKind)

		//Query and store prometheus Memory requests
		query = `avg(sum(kube_pod_container_resource_requests_memory_bytes/1024/1024) by (node) / sum(kube_node_status_capacity_memory_bytes/1024/1024) by (node)` + nodeGroupSuffix + ` * 100`
		getWorkload("memory_reservation_percent", "MemoryReservationPercent", query, nodeGroupLabels, args, entityKind)
	}

	//Check to see which disk queries to use if instance is IP address that need to link to pod to get name or if instance = node name.
	query = `max(max(label_replace(sum(irate(node_cpu_seconds_total{mode!="idle"}[` + args.SampleRateString + `m])) by (instance) / on (instance) group_left count(node_cpu_seconds_total{mode="idle"}) by (instance) *100, "pod_ip", "$1", "instance", "(.*):.*")) by (pod_ip) * on (pod_ip) group_right kube_pod_info{pod=~".*node-exporter.*"}) by (node)`
	result, err = common.MetricCollect(args, query, range5Min)

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

	query = `sum(kube_node_labels{stringToBeReplaced=~".+"}) by (stringToBeReplaced)`
	getWorkload("current_size", "CurrentSize", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus total cpu uptime in seconds
	query = queryPrefix + `sum(irate(node_cpu_seconds_total{mode!="idle"}[` + args.SampleRateString + `m])) by (instance) / on (instance) group_left count(node_cpu_seconds_total{mode="idle"}) by (instance) *100` + querySuffix
	getWorkload("cpu_utilization", "CpuUtilization", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus node memory total in bytes
	query = queryPrefix + `node_memory_MemTotal_bytes - node_memory_MemFree_bytes` + querySuffix
	getWorkload("memory_raw_bytes", "MemoryBytes", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus node memory total free in bytes
	query = queryPrefix + `node_memory_MemTotal_bytes - (node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes)` + querySuffix
	getWorkload("memory_actual_workload", "MemoryActualWorkload", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus node disk write in bytes
	query = queryPrefixSum + `irate(node_disk_written_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("disk_write_bytes", "DiskWriteBytes", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus node disk read in bytes
	query = queryPrefixSum + `irate(node_disk_read_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("disk_read_bytes", "DiskReadBytes", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus total disk read uptime as a percentage
	query = queryPrefixSum + `irate(node_disk_read_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("disk_read_ops", "DiskReadOps", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus total disk write uptime as a percentage
	query = queryPrefixSum + `irate(node_disk_write_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("disk_write_ops", "DiskWriteOps", query, nodeGroupLabels, args, entityKind)

	//Total disk values
	//Query and store prometheus node disk read in bytes
	query = queryPrefixSum + `irate(node_disk_read_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) + irate(node_disk_written_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("disk_total_bytes", "DiskTotalBytes", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus total disk read uptime as a percentage
	query = queryPrefixSum + `(irate(node_disk_read_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) + irate(node_disk_write_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("disk_total_ops", "DiskTotalOps", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus node recieved network data in bytes
	query = queryPrefixSum + `irate(node_network_receive_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("net_received_bytes", "NetReceivedBytes", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus recieved network data in packets
	query = queryPrefixSum + `irate(node_network_receive_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("net_received_packets", "NetReceivedPackets", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus total transmitted network data in bytes
	query = queryPrefixSum + `irate(node_network_transmit_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("net_sent_bytes", "NetSentBytes", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus total transmitted network data in packets
	query = queryPrefixSum + `irate(node_network_transmit_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("net_sent_packets", "NetSentPackets", query, nodeGroupLabels, args, entityKind)

	//Total values network
	//Query and store prometheus total network data in bytes
	query = queryPrefixSum + `irate(node_network_transmit_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m]) + irate(node_network_receive_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("net_total_bytes", "NetTotalBytes", query, nodeGroupLabels, args, entityKind)

	//Query and store prometheus total network data in packets
	query = queryPrefixSum + `irate(node_network_transmit_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m]) + irate(node_network_receive_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])` + querySuffixSum
	getWorkload("net_total_packets", "NetTotalPackets", query, nodeGroupLabels, args, entityKind)
}
