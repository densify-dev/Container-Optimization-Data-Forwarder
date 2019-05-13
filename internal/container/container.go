//Package container collects data related to containers and formats into csv files to send to Densify.
package container

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//Global variables used for Storing system info, command line\config file parameters.
var systems = map[string]*namespace{}

//container is used to hold information related to the containers defined in a pod
type container struct {
	memory, cpuLimit, cpuRequest, memLimit, memRequest, restarts, powerState int
	conLabel, conInfo, currentNodes, podName                                 string
}

//pod is used to hold information related to the controllers or individual pods in a namespace.
type pod struct {
	podInfo, podLabel, ownerKind, ownerName, controllerLabel string
	currentSize                                              int
	creationTime                                             int64
	containers                                               map[string]*container
}

//namespace is used to hold information related to the namespaces defined in Kubernetes
type namespace struct {
	namespaceLabel                             string
	cpuLimit, cpuRequest, memLimit, memRequest int
	pods                                       map[string]*pod
}

//Metrics function to collect data related to containers.
func Metrics(clusterName, promProtocol, promAddr, promPort, interval string, intervalSize, history int, debug bool, currentTime time.Time) {

	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var query, promaddress string
	var result model.Value
	var start, end time.Time

	start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
	promaddress = promProtocol + "://" + promAddr + ":" + promPort

	// For the queries we have throughout you will see each query is called twice with minor tweaks this is cause we query once to get all the containers that are part of controllers and a second time to get all the containers that are setup as individual pods.
	//This was done as the label we use for controller based (owner_name) is set to be <none> for all the individual pods and if we query them together for certain fields it would combine values\labels of the individual pods so you would see tags that aren't actually on your container.

	//Query for memory limit set for containers. This query is for the controller based pods.
	query = `max(sum(container_spec_memory_limit_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)/1024/1024 * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (owner_name,owner_kind,namespace,container_name)`
	result = prometheus.MetricCollect(promaddress, query, start, end)

	//setup the system data structure for new systems and load existing ones.
	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure and if not add it.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric["namespace"]; ok {
				if _, ok := systems[string(namespaceValue)]; ok == false {
					systems[string(namespaceValue)] = &namespace{namespaceLabel: "", cpuRequest: -1, cpuLimit: -1, memRequest: -1, memLimit: -1, pods: map[string]*pod{}}
				}
				//Validate that the data contains the owner_name label (This will be the pod field when writing out data) with value and check it exists in our systems structure and if not add it.
				if ownerValue, ok := result.(model.Matrix)[i].Metric["owner_name"]; ok {
					if _, ok := systems[string(namespaceValue)].pods[string(ownerValue)]; ok == false {
						if ownerKind, ok := result.(model.Matrix)[i].Metric["owner_kind"]; ok {
							systems[string(namespaceValue)].pods[string(ownerValue)] = &pod{podInfo: "", podLabel: "", ownerKind: string(ownerKind), ownerName: string(ownerValue), controllerLabel: "", creationTime: -1, currentSize: -1, containers: map[string]*container{}}
						} else {
							systems[string(namespaceValue)].pods[string(ownerValue)] = &pod{podInfo: "", podLabel: "", ownerKind: "", ownerName: string(ownerValue), controllerLabel: "", creationTime: -1, currentSize: -1, containers: map[string]*container{}}
						}
					}
					//Validate that the data contains the container label with value and check it exists in our systems structure and if not add it
					if containerValue, ok := result.(model.Matrix)[i].Metric["container_name"]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(ownerValue)].containers[string(containerValue)]; ok == false {
							var memSize int
							//Check the length of the values array if it is empty then set memory to 0 otherwise use the last\current value in the array as the size of the memory.
							if len(result.(model.Matrix)[i].Values) == 0 {
								memSize = 0
							} else {
								memSize = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
							}
							systems[string(namespaceValue)].pods[string(ownerValue)].containers[string(containerValue)] = &container{memory: memSize, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, restarts: -1, powerState: 1, conLabel: "", conInfo: "", currentNodes: "", podName: ""}
						}
					}
				}
			}
		}
	}

	//If debuging on write out the current systems data.
	if debug {
		log.Println("[DEBUG] Output of systems after initial call to setup controllers")
		log.Println("[DEBUG] ", systems)
	}

	//Query for memory limit set for containers. This query is for the individual based pods.
	query = `max(sum(container_spec_memory_limit_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)/1024/1024 * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")) by (pod_name,namespace,container_name)`
	result = prometheus.MetricCollect(promaddress, query, start, end)

	if result != nil {
		//Loop through the different entities in the results.
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			//Validate that the data contains the namespace label with value and check it exists in our systems structure and if not add it.
			if namespaceValue, ok := result.(model.Matrix)[i].Metric["namespace"]; ok {
				if _, ok := systems[string(namespaceValue)]; ok == false {
					systems[string(namespaceValue)] = &namespace{namespaceLabel: "", cpuRequest: -1, cpuLimit: -1, memRequest: -1, memLimit: -1, pods: map[string]*pod{}}
				}
				//Validate that the data contains the pod_name label (This will be the pod field when writing out data) with value and check it exists in our systems structure and if not add it.
				if podValue, ok := result.(model.Matrix)[i].Metric["pod_name"]; ok {
					if _, ok := systems[string(namespaceValue)].pods[string(podValue)]; ok == false {
						systems[string(namespaceValue)].pods[string(podValue)] = &pod{podInfo: "", podLabel: "", ownerKind: "<none>", ownerName: string(podValue), controllerLabel: "", creationTime: -1, currentSize: -1, containers: map[string]*container{}}
					}
					//Validate that the data contains the container label with value and check it exists in our systems structure and if not add it
					if containerValue, ok := result.(model.Matrix)[i].Metric["container_name"]; ok {
						if _, ok := systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)]; ok == false {
							var memSize int
							//Check the length of the values array if it is empty then set memory to 0 otherwise use the last\current value in the array as the size of the memory.
							if len(result.(model.Matrix)[i].Values) == 0 {
								memSize = 0
							} else {
								memSize = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
							}
							systems[string(namespaceValue)].pods[string(podValue)].containers[string(containerValue)] = &container{memory: memSize, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, restarts: -1, powerState: 1, conLabel: "", conInfo: "", currentNodes: "", podName: ""}
						}
					}
				}
			}
		}
	}
	//validate there are systems created from 1 of the 2 queries above and if not log error to validate the Prometheus settings and exit.
	if len(systems) == 0 {
		fmt.Println("No data returned from Prometheus. Validate all the prerequisites are setup")
		log.Fatalln("No data returned from Prometheus. Validate all the prerequisites are setup")
	}

	//Write out the systems data structure if debug is enabled.
	if debug {
		log.Println("[DEBUG] Output of systems after initial call to setup individual pods")
		log.Println("[DEBUG] ", systems)
	}

	//variables that were used in prometheus to simplify the repetitive code.
	var kubeStateOwner, kubeStatePod string
	kubeStateOwner = ` * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}) by (owner_name,owner_kind,namespace,container)`
	kubeStatePod = ` * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}) by (pod,namespace,container)`

	//Container metrics
	//Get the CPU Limit for container
	query = `max(sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000` + kubeStateOwner
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "owner_name", "container", "cpuLimit")

	query = `max(sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000` + kubeStatePod
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "cpuLimit")

	//Get the CPU Request for container
	query = `max(sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000` + kubeStateOwner
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "owner_name", "container", "cpuRequest")

	query = `max(sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000` + kubeStatePod
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "cpuRequest")

	//Get the Memory Limit for container
	query = `max(sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024` + kubeStateOwner
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "owner_name", "container", "memLimit")

	query = `max(sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024` + kubeStatePod
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "memLimit")

	//Get the Memory Request for container
	query = `max(sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024` + kubeStateOwner
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "owner_name", "container", "memRequest")

	query = `max(sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024` + kubeStatePod
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "memRequest")

	//Get the number of times the container has been restarted
	query = `max(sum(kube_pod_container_status_restarts_total) by (pod,namespace,container)` + kubeStateOwner
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "owner_name", "container", "restarts")

	query = `max(sum(kube_pod_container_status_restarts_total) by (pod,namespace,container)` + kubeStatePod
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "restarts")

	//Check to see if the container is still running or if it has been terminated.
	query = `max(sum(kube_pod_container_status_terminated) by (pod,namespace,container)` + kubeStateOwner
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "owner_name", "container", "powerState")

	query = `max(sum(kube_pod_container_status_terminated) by (pod,namespace,container)` + kubeStatePod
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "powerState")

	//Get the container labels.
	query = `(sum(container_spec_cpu_shares{name!~"k8s_POD_.*"}) by (pod_name,namespace,container_name)) * on (namespace,pod_name,container_name) group_right container_spec_cpu_shares * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind!="<none>"}, "pod_name", "$1", "pod", "(.*)")`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetricString(result, "namespace", "owner_name", "container_name", "conLabel")

	query = `(sum(container_spec_cpu_shares{name!~"k8s_POD_.*"}) by (pod_name,namespace,container_name)) * on (namespace,pod_name,container_name) group_right container_spec_cpu_shares * on (namespace,pod_name) group_left (owner_name,owner_kind) label_replace(kube_pod_owner{owner_kind="<none>"}, "pod_name", "$1", "pod", "(.*)")`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetricString(result, "namespace", "pod_name", "container_name", "conLabel")

	//Get the container info values.
	query = `sum(kube_pod_container_info) by (pod,namespace,container) * on (namespace,pod,container) group_right kube_pod_container_info * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetricString(result, "namespace", "owner_name", "container", "conInfo")

	query = `sum(kube_pod_container_info) by (pod,namespace,container) * on (namespace,pod,container) group_right kube_pod_container_info * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetricString(result, "namespace", "pod", "container", "conInfo")

	//Pod Metrics
	//Get the pod info values
	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_owner{owner_kind!="<none>"}`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetricString(result, "namespace", "owner_name", "podInfo")

	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_owner{owner_kind="<none>"}`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetricString(result, "namespace", "pod", "podInfo")

	//Get the pod labels.
	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_labels * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind!="<none>"}`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetricString(result, "namespace", "owner_name", "podLabel")

	query = `sum(kube_pod_container_info) by (pod,namespace) * on (namespace,pod) group_right kube_pod_labels * on (namespace,pod) group_left (owner_name,owner_kind) kube_pod_owner{owner_kind="<none>"}`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetricString(result, "namespace", "pod", "podLabel")

	currentSizeWrite, err := os.Create("./data/currentSize.csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(currentSizeWrite, "cluster,namespace,pod,container,Datetime,currentSize\n")

	//Get the current size of the controller will query each of the differnt types of controller
	query = `kube_replicaset_spec_replicas`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetric(result, "namespace", "replicaset", "currentSize")
	writeWorkloadPod(currentSizeWrite, result, "namespace", "replicaset", clusterName, promAddr)

	query = `kube_replicationcontroller_spec_replicas`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetric(result, "namespace", "replicationcontroller", "currentSize")
	writeWorkloadPod(currentSizeWrite, result, "namespace", "replicationcontroller", clusterName, promAddr)

	query = `kube_daemonset_status_number_available`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetric(result, "namespace", "daemonset", "currentSize")
	writeWorkloadPod(currentSizeWrite, result, "namespace", "daemonset", clusterName, promAddr)

	query = `kube_statefulset_replicas`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetric(result, "namespace", "statefulset", "currentSize")
	writeWorkloadPod(currentSizeWrite, result, "namespace", "statefulset", clusterName, promAddr)

	query = `kube_job_spec_parallelism`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetric(result, "namespace", "job_name", "currentSize")
	writeWorkloadPod(currentSizeWrite, result, "namespace", "job_name", clusterName, promAddr)

	currentSizeWrite.Close()

	//Get the controller labels
	query = `kube_statefulset_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetricString(result, "namespace", "statefulset", "controllerLabel")

	query = `kube_job_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetricString(result, "namespace", "job_name", "controllerLabel")

	query = `kube_daemonset_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetricString(result, "namespace", "daemonset", "controllerLabel")

	//Get when the pod was originally created.
	query = `max(kube_pod_created` + kubeStateOwner
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetric(result, "namespace", "owner_name", "creationTime")

	query = `max(kube_pod_created` + kubeStatePod
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getPodMetric(result, "namespace", "pod", "creationTime")

	//Namespace Metrics
	//Get the namespace labels
	query = `kube_namespace_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getNamespaceMetricString(result, "namespace", "namespaceLabel")

	//Get the CPU and Memory Limit and Request quotes for the namespace.
	query = `kube_limitrange`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getNamespacelimits(result, "namespace")

	//Write out the config and attributes files.
	writeConfig(clusterName, promAddr)
	writeAttributes(clusterName, promAddr)

	query = `round(sum(rate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)*1000,1)`
	getWorkload(promaddress, "cpu_mCores_workload", "CPU Utilization in mCores", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `sum(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload(promaddress, "mem_workload", "Raw Mem Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `sum(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload(promaddress, "rss_workload", "Actual Memory Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `sum(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload(promaddress, "disk_workload", "Raw Disk Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `round(sum(rate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)*1000,1)`
	getWorkload(promaddress, "cpu_mCores_workload", "CPU Utilization in mCores", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `sum(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload(promaddress, "mem_workload", "Raw Mem Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `sum(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload(promaddress, "rss_workload", "Actual Memory Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `sum(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)`
	getWorkload(promaddress, "disk_workload", "Raw Disk Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)
}
