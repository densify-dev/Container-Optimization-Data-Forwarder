//Package cluster collects data related to containers and formats into csv files to send to Densify.
package cluster

import (
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
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

//Gets cluster metrics from prometheus (and checks to see if they are valid)
func getClusterMetric(result model.Value, metric string) {

	//validates that the value of the entity is set and if not will default to 0
	var value int
	if len(result.(model.Matrix)[0].Values) == 0 {
		value = 0
	} else {
		value = int(result.(model.Matrix)[0].Values[len(result.(model.Matrix)[0].Values)-1].Value)
	}

	//Check which metric this is for and update the corresponding variable for this container in the system data structure

	switch metric {
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

func getWorkload(fileName, metricName, query string, args *common.Parameters) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value

	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/cluster/" + fileName + ".csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " metric=" + metricName + " query=" + query + " message=" + err.Error())
		fmt.Println("entity=" + entityKind + " metric=" + metricName + " query=" + query + " message=" + err.Error())
		return
	}
	fmt.Fprintf(workloadWrite, "cluster,Datetime,%s\n", metricName)

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slpwer prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < *args.History; historyInterval++ {
		range5Min := prometheus.TimeRange(args, historyInterval)

		result = prometheus.MetricCollect(args, query, range5Min, metricName, false)
		if result != nil {
			writeWorkload(workloadWrite, result, args)
		}
	}
	//Close the workload files.
	workloadWrite.Close()
}

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkload(file io.Writer, result model.Value, args *common.Parameters) {
	//Loop through the different values over the interval and write out each one to the workload file.
	for j := 0; j < len(result.(model.Matrix)[0].Values); j++ {
		var val model.SampleValue
		if math.IsNaN(float64(result.(model.Matrix)[0].Values[j].Value)) || math.IsInf(float64(result.(model.Matrix)[0].Values[j].Value), 0) {
			val = 0
		} else {
			val = result.(model.Matrix)[0].Values[j].Value
		}
		fmt.Fprintf(file, "%s,%s,%f\n",
			*args.ClusterName,
			time.Unix(0, int64(result.(model.Matrix)[0].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"),
			val)
	}

}

//writeConfig will create the config.csv file that is will be sent Densify by the Forwarder.
func writeConfig(args *common.Parameters) {

	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/cluster/config.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster")
	fmt.Fprintf(configWrite, "%s\n", *args.ClusterName)
	configWrite.Close()
}

//writeAttributes will create the attributes.csv file that is will be sent Densify by the Forwarder.
func writeAttributes(args *common.Parameters) {

	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/cluster/attributes.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,Virtual Technology,Virtual Domain,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request")

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
	var query string
	var result model.Value

	//Start and end time + the prometheus address used for querying
	range5Min := prometheus.TimeRange(args, historyInterval)

	query = `sum(kube_pod_container_resource_limits_cpu_cores*1000 * on (namespace,pod,container) group_left kube_pod_container_status_running)`
	result = prometheus.MetricCollect(args, query, range5Min, "cpuLimit", false)
	if result != nil {
		getClusterMetric(result, "cpuLimit")
	}

	query = `sum(kube_pod_container_resource_requests_cpu_cores*1000 * on (namespace,pod,container) group_left kube_pod_container_status_running)`
	result = prometheus.MetricCollect(args, query, range5Min, "cpuRequest", false)
	if result != nil {
		getClusterMetric(result, "cpuRequest")
	}

	query = `sum(kube_pod_container_resource_limits_memory_bytes/1024/1024 * on (namespace,pod,container) group_left kube_pod_container_status_running)`
	result = prometheus.MetricCollect(args, query, range5Min, "memLimit", false)
	if result != nil {
		getClusterMetric(result, "memLimit")
	}

	query = `sum(kube_pod_container_resource_requests_memory_bytes/1024/1024 * on (namespace,pod,container) group_left kube_pod_container_status_running)`
	result = prometheus.MetricCollect(args, query, range5Min, "memRequest", false)
	if result != nil {
		getClusterMetric(result, "memRequest")
	}

	writeAttributes(args)
	writeConfig(args)

	//Query and store prometheus CPU requests
	query = `sum((kube_pod_container_resource_requests_cpu_cores) * on (namespace,pod,container) group_left kube_pod_container_status_running)`
	getWorkload("cpu_requests", "CPU Reservation in Cores", query, args)

	//Query and store prometheus CPU requests
	query = `sum((kube_pod_container_resource_requests_cpu_cores) * on (namespace,pod,container) group_left kube_pod_container_status_running) / sum(kube_node_status_allocatable_cpu_cores) * 100`
	getWorkload("cpu_reservation_percent", "CPU Reservation Percent", query, args)

	//Query and store prometheus Memory requests
	query = `sum((kube_pod_container_resource_requests_memory_bytes/1024/1024) * on (namespace,pod,container) group_left kube_pod_container_status_running)`
	getWorkload("memory_requests", "Memory Reservation in MB", query, args)

	//Query and store prometheus Memory requests
	query = `sum((kube_pod_container_resource_requests_memory_bytes/1024/1024) * on (namespace,pod,container) group_left kube_pod_container_status_running) / sum(kube_node_status_allocatable_memory_bytes/1024/1024) * 100`
	getWorkload("memory_reservation_percent", "Memory Reservation Percent", query, args)
}
