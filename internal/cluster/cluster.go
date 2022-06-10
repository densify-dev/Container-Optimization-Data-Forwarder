//Package cluster collects data related to containers and formats into csv files to send to Densify.
package cluster

import (
	"fmt"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
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
var entityKind = "cluster"

//Gets cluster metrics from prometheus (and checks to see if they are valid)
func getClusterMetric(result model.Value, metric string) {

	if mat, ok := result.(model.Matrix); ok {
		for i := 0; i < mat.Len(); i++ {
			//validates that the value of the entity is set and if not will default to 0
			var value float64
			if len(mat[i].Values) == 0 {
				value = 0
			} else {
				value = float64(mat[i].Values[len(mat[i].Values)-1].Value)
			}

			//Check which metric this is for and update the corresponding variable for this container in the system data structure

			switch metric {
			case "limits":
				switch mat[i].Metric["resource"] {
				case "memory":
					clusterEntity.memLimit = int(value / 1024 / 1024)
				case "cpu":
					clusterEntity.cpuLimit = int(value * 1000)
				}
			case "requests":
				switch mat[i].Metric["resource"] {
				case "memory":
					clusterEntity.memRequest = int(value / 1024 / 1024)
				case "cpu":
					clusterEntity.cpuRequest = int(value * 1000)
				}
			case "cpuLimit":
				clusterEntity.cpuLimit = int(value)
			case "cpuRequest":
				clusterEntity.cpuRequest = int(value)
			case "memLimit":
				clusterEntity.memLimit = int(value)
			case "memRequest":
				clusterEntity.memRequest = int(value)
			}
		}
	}

}

//writeConfig will create the config.csv file that is will be sent to Densify by the Forwarder.
func writeConfig(args *common.Parameters) {

	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/cluster/config.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "AuditTime,Name")
	fmt.Fprintf(configWrite, "%s,%s\n", common.Format(args.CurrentTime), *args.ClusterName)
	configWrite.Close()
}

//writeAttributes will create the attributes.csv file that is will be sent to Densify by the Forwarder.
func writeAttributes(args *common.Parameters) {

	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/cluster/attributes.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "Name,VirtualTechnology,VirtualDomain,CpuLimit,CpuRequest,MemoryLimit,MemoryRequest")

	//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
	fmt.Fprintf(attributeWrite, "%s,Clusters,%s", *args.ClusterName, *args.ClusterName)

	if clusterEntity.cpuLimit == -1 {
		fmt.Fprintf(attributeWrite, ",")
	} else {
		fmt.Fprintf(attributeWrite, ",%d", clusterEntity.cpuLimit)
	}

	if clusterEntity.cpuRequest == -1 {
		fmt.Fprintf(attributeWrite, ",")
	} else {
		fmt.Fprintf(attributeWrite, ",%d", clusterEntity.cpuRequest)
	}

	if clusterEntity.memLimit == -1 {
		fmt.Fprintf(attributeWrite, ",")
	} else {
		fmt.Fprintf(attributeWrite, ",%d", clusterEntity.memLimit)
	}

	if clusterEntity.memRequest == -1 {
		fmt.Fprintf(attributeWrite, ",\n")
	} else {
		fmt.Fprintf(attributeWrite, ",%d\n", clusterEntity.memRequest)
	}

	attributeWrite.Close()
}

//Metrics a global func for collecting node level metrics in prometheus
func Metrics(args *common.Parameters) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var query, requestsLabel string
	var result model.Value
	var err error

	//Start and end time + the prometheus address used for querying
	range5Min := common.TimeRange(args, historyInterval)

	query = `sum(kube_pod_container_resource_limits) by (resource)`
	result, err = common.MetricCollect(args, query, range5Min)
	if mat, ok := result.(model.Matrix); err != nil || !ok || mat.Len() == 0 {
		query = `sum(kube_pod_container_resource_limits_cpu_cores*1000)`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=cpuLimit query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cpuLimit query=" + query + " message=" + err.Error())
		} else {
			getClusterMetric(result, "cpuLimit")
		}

		query = `sum(kube_pod_container_resource_limits_memory_bytes/1024/1024)`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=memLimit query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=memLimit query=" + query + " message=" + err.Error())
		} else {
			getClusterMetric(result, "memLimit")
		}
	} else {
		getClusterMetric(result, "limits")
	}

	query = `sum(kube_pod_container_resource_requests) by (resource)`
	result, err = common.MetricCollect(args, query, range5Min)
	if mat, ok := result.(model.Matrix); err != nil || !ok || mat.Len() == 0 {
		query = `sum(kube_pod_container_resource_requests_cpu_cores*1000)`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=cpuRequest query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cpuRequest query=" + query + " message=" + err.Error())
		} else {
			getClusterMetric(result, "cpuRequest")
		}

		query = `sum(kube_pod_container_resource_requests_memory_bytes/1024/1024)`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=memRequest query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=memRequest query=" + query + " message=" + err.Error())
		} else {
			getClusterMetric(result, "memRequest")
		}
	} else {
		getClusterMetric(result, "requests")
		requestsLabel = "unified"
	}

	writeAttributes(args)
	writeConfig(args)

	var metricField []model.LabelName

	if requestsLabel == "unified" {
		//Query and store prometheus CPU requests
		query = `avg(sum(kube_pod_container_resource_requests{resource="cpu"}) by (node))`
		common.GetWorkload("cpu_requests", "CpuRequests", query, metricField, args, entityKind)

		//Query and store prometheus CPU requests
		query = `avg(sum(kube_pod_container_resource_requests{resource="cpu"}) by (node) / sum(kube_node_status_capacity{resource="cpu"}) by (node)) * 100`
		common.GetWorkload("cpu_reservation_percent", "CpuReservationPercent", query, metricField, args, entityKind)

		//Query and store prometheus Memory requests
		query = `avg(sum(kube_pod_container_resource_requests{resource="memory"}/1024/1024) by (node))`
		common.GetWorkload("memory_requests", "MemoryRequests", query, metricField, args, entityKind)

		//Query and store prometheus Memory requests
		query = `avg(sum(kube_pod_container_resource_requests{resource="memory"}/1024/1024) by (node) / sum(kube_node_status_capacity{resource="memory"}/1024/1024) by (node)) * 100`
		common.GetWorkload("memory_reservation_percent", "MemoryReservationPercent", query, metricField, args, entityKind)
	} else {
		//Query and store prometheus CPU requests
		query = `avg(sum(kube_pod_container_resource_requests_cpu_cores) by (node))`
		common.GetWorkload("cpu_requests", "CpuRequests", query, metricField, args, entityKind)

		//Query and store prometheus CPU requests
		query = `avg(sum(kube_pod_container_resource_requests_cpu_cores) by (node) / sum(kube_node_status_capacity_cpu_cores) by (node)) * 100`
		common.GetWorkload("cpu_reservation_percent", "CpuReservationPercent", query, metricField, args, entityKind)

		//Query and store prometheus Memory requests
		query = `avg(sum(kube_pod_container_resource_requests_memory_bytes/1024/1024) by (node))`
		common.GetWorkload("memory_requests", "MemoryRequests", query, metricField, args, entityKind)

		//Query and store prometheus Memory requests
		query = `avg(sum(kube_pod_container_resource_requests_memory_bytes/1024/1024) by (node) / sum(kube_node_status_capacity_memory_bytes/1024/1024) by (node)) * 100`
		common.GetWorkload("memory_reservation_percent", "MemoryReservationPercent", query, metricField, args, entityKind)
	}

	//For cluster we don't have to check instance field and convert to pod_ip as we aren't looking to map to the node names but rather just get the avg for nodes. So we can use just instance field in all cases.
	//Query and store prometheus total cpu uptime in seconds
	query = `avg(sum(irate(node_cpu_seconds_total{mode!="idle"}[` + args.SampleRateString + `m])) by (instance) / on (instance) group_left count(node_cpu_seconds_total{mode="idle"}) by (instance) *100)`
	common.GetWorkload("cpu_utilization", "CpuUtilization", query, metricField, args, entityKind)

	//Query and store prometheus node memory total in bytes
	query = `avg(node_memory_MemTotal_bytes - node_memory_MemFree_bytes)`
	common.GetWorkload("memory_raw_bytes", "MemoryBytes", query, metricField, args, entityKind)

	//Query and store prometheus node memory total free in bytes
	query = `avg(node_memory_MemTotal_bytes - (node_memory_MemFree_bytes + node_memory_Cached_bytes + node_memory_Buffers_bytes))`
	common.GetWorkload("memory_actual_workload", "MemoryActualWorkload", query, metricField, args, entityKind)

	//Query and store prometheus node disk write in bytes
	query = `avg(sum(irate(node_disk_written_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("disk_write_bytes", "DiskWriteBytes", query, metricField, args, entityKind)

	//Query and store prometheus node disk read in bytes
	query = `avg(sum(irate(node_disk_read_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("disk_read_bytes", "DiskReadBytes", query, metricField, args, entityKind)

	//Query and store prometheus total disk read uptime as a percentage
	query = `avg(sum(irate(node_disk_read_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("disk_read_ops", "DiskReadOps", query, metricField, args, entityKind)

	//Query and store prometheus total disk write uptime as a percentage
	query = `avg(sum(irate(node_disk_write_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("disk_write_ops", "DiskWriteOps", query, metricField, args, entityKind)

	//Total disk values
	//Query and store prometheus node disk read in bytes
	query = `avg(sum(irate(node_disk_read_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) + irate(node_disk_written_bytes_total{device!~"dm-.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("disk_total_bytes", "DiskTotalBytes", query, metricField, args, entityKind)

	//Query and store prometheus total disk read uptime as a percentage
	query = `avg(sum((irate(node_disk_read_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m]) + irate(node_disk_write_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])) / irate(node_disk_io_time_seconds_total{device!~"dm-.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("disk_total_ops", "DiskTotalOps", query, metricField, args, entityKind)

	//Query and store prometheus node received network data in bytes
	query = `avg(sum(irate(node_network_receive_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("net_received_bytes", "NetReceivedBytes", query, metricField, args, entityKind)

	//Query and store prometheus recieved network data in packets
	query = `avg(sum(irate(node_network_receive_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("net_received_packets", "NetReceivedPackets", query, metricField, args, entityKind)

	//Query and store prometheus total transmitted network data in bytes
	query = `avg(sum(irate(node_network_transmit_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("net_sent_bytes", "NetSentBytes", query, metricField, args, entityKind)

	//Query and store prometheus total transmitted network data in packets
	query = `avg(sum(irate(node_network_transmit_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("net_sent_packets", "NetSentPackets", query, metricField, args, entityKind)

	//Total values network
	//Query and store prometheus total network data in bytes
	query = `avg(sum(irate(node_network_transmit_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m]) + irate(node_network_receive_bytes_total{device!~"veth.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("net_total_bytes", "NetTotalBytes", query, metricField, args, entityKind)

	//Query and store prometheus total network data in packets
	query = `avg(sum(irate(node_network_transmit_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m]) + irate(node_network_receive_packets_total{device!~"veth.*"}[` + args.SampleRateString + `m])) by (instance))`
	common.GetWorkload("net_total_packets", "NetTotalPackets", query, metricField, args, entityKind)

}
