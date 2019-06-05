//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"os"
	"log"
	"fmt"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

var namespaces = map[string]*namespace{}

type namespace struct {
	namespaceLabel                             string
	pointers                                   map[string]*topLevel
	topLevels                                  map[string]*topLevel
	cpuLimit, cpuRequest, memLimit, memRequest int
}

//topLevel is used to hold information related to the highest owner of any containers
type topLevel struct {
	name, kind, namespace               string
	containers                          map[string]*container
	currentSize, restarts int
	creationTime int64
	labelMap                            map[string]string
}

//container is used to hold information related to containers
type container struct {
	memory, cpuLimit, cpuRequest, memLimit, memRequest, restarts, powerState        int
	name, conLabel, conInfo, currentNodes, namespaceLabel, controllerLabel, podName string
	labelMap                                                                        map[string]string
}

//Metrics function to collect data related to containers.
func Metrics(clusterName, promProtocol, promAddr, promPort, interval string, intervalSize, history int, debug bool, currentTime time.Time) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var query1, query2, query3, query4, promaddress string
	var result, result1, result2, result3, result4 model.Value
	var start, end time.Time

	_ = result

	start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
	promaddress = promProtocol + "://" + promAddr + ":" + promPort

	//querys gathering hierarchy information for the containers
	query1 = `max(kube_pod_container_info) by (container, pod, namespace)`
	query2 = `kube_pod_owner{owner_name!="<none>"}`
	query3 = `kube_replicaset_owner{owner_name!="<none>"}`
	query4 = `kube_job_owner{owner_name!="<none>"}`

	//results stored in variables
	result1 = prometheus.MetricCollect(promaddress, query1, start, end)
	result2 = prometheus.MetricCollect(promaddress, query2, start, end)
	result3 = prometheus.MetricCollect(promaddress, query3, start, end)
	result4 = prometheus.MetricCollect(promaddress, query4, start, end)

	//maps of names corresponding to owner information
	var podNamespaces = map[string]string{}
	var podOwners = map[string]string{}
	var podOwnersKind = map[string]string{}
	var replicaSetOwners = map[string]string{}
	var jobOwners = map[string]string{}

	//variables to shorten typing out results
	var containerRslt = result1.(model.Matrix)
	var podRslt = result2.(model.Matrix)
	var replicaSetRslt = result3.(model.Matrix)
	var jobRslt = result4.(model.Matrix)
	var currentOwner string

	//use the pod name to store the namespace the pod resides in
	for i := 0; i < containerRslt.Len(); i++ {
		podNamespaces[string(containerRslt[i].Metric["pod"])] = string(containerRslt[i].Metric["namespace"])
	}

	//use the pod name to store namespace, owner_name and owner_kind into respective maps
	for i := 0; i < podRslt.Len(); i++ {
		podNamespaces[string(podRslt[i].Metric["pod"])] = string(podRslt[i].Metric["namespace"])
		podOwners[string(podRslt[i].Metric["pod"])] = string(podRslt[i].Metric["owner_name"])
		podOwnersKind[string(podRslt[i].Metric["pod"])] = string(podRslt[i].Metric["owner_kind"])
	}

	//uses replicasets as key to store
	for i := 0; i < replicaSetRslt.Len(); i++ {
		replicaSetOwners[string(replicaSetRslt[i].Metric["replicaset"])] = string(replicaSetRslt[i].Metric["owner_name"])
	}

	//uses jobs as ke to store
	for i := 0; i < jobRslt.Len(); i++ {
		jobOwners[string(jobRslt[i].Metric["job_name"])] = string(jobRslt[i].Metric["owner_name"])
	}
	//Add containers and top owners to structure
	for i := 0; i < containerRslt.Len(); i++ {

		containerName := string(containerRslt[i].Metric["container"])
		podName := string(containerRslt[i].Metric["pod"])
		var ownerKind string

		namespaceName := podNamespaces[podName]
		if _, ok := namespaces[namespaceName]; !ok {
			namespaces[namespaceName] = &namespace{pointers: map[string]*topLevel{}, topLevels: map[string]*topLevel{}, cpuRequest: -1, cpuLimit: -1, memRequest: -1, memLimit: -1}
		}

		//namespaces[namespaceName].pods[podName] = &pod{labelMap: map[string]string{}}
		if controllerName, ok := podOwners[podName]; ok {
			if deploymentName, ok := replicaSetOwners[controllerName]; ok && podOwnersKind[podName] == "ReplicaSet" {
				currentOwner = deploymentName
				ownerKind = "Deployment"
				//Create deployment as top owner and add container
				if _, ok := namespaces[namespaceName].topLevels["Deployment__"+deploymentName]; !ok {
					namespaces[namespaceName].topLevels["Deployment__"+deploymentName] = &topLevel{name: deploymentName, kind: "Deployment", containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
					namespaces[namespaceName].topLevels["Deployment__"+deploymentName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}}
				} else if _, ok := namespaces[namespaceName].topLevels["Deployment__"+deploymentName].containers[containerName]; !ok {
					namespaces[namespaceName].topLevels["Deployment__"+deploymentName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}}
				}
				if _, ok := namespaces[namespaceName].pointers["Deployment__"+deploymentName]; !ok {
					namespaces[namespaceName].pointers["Deployment__"+deploymentName] = namespaces[namespaceName].topLevels[ownerKind+"__"+currentOwner]
				}
			} else if cronJobName, ok := jobOwners[controllerName]; ok && podOwnersKind[podName] == "Job" {
				currentOwner = cronJobName
				ownerKind = "CronJob"
				//Create deployment as top owner and add container
				if _, ok := namespaces[namespaceName].topLevels["CronJob__"+cronJobName]; !ok {
					namespaces[namespaceName].topLevels["CronJob__"+cronJobName] = &topLevel{name: cronJobName, kind: "CronJob", containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
					namespaces[namespaceName].topLevels["CronJob__"+cronJobName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}}
				} else if _, ok := namespaces[namespaceName].topLevels["CronJob__"+cronJobName].containers[containerName]; !ok {
					namespaces[namespaceName].topLevels["CronJob__"+cronJobName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}}
				}
				if _, ok := namespaces[namespaceName].pointers["CronJob__"+cronJobName]; !ok {
					namespaces[namespaceName].pointers["CronJob__"+cronJobName] = namespaces[namespaceName].topLevels[ownerKind+"__"+currentOwner]
				}
			} else {
				currentOwner = controllerName
				ownerKind = podOwnersKind[podName]
				//Create controller as top owner and add container
				if _, ok := namespaces[namespaceName].topLevels[podOwnersKind[podName]+"__"+controllerName]; !ok {
					namespaces[namespaceName].topLevels[podOwnersKind[podName]+"__"+controllerName] = &topLevel{name: controllerName, kind: podOwnersKind[podName], containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
					namespaces[namespaceName].topLevels[podOwnersKind[podName]+"__"+controllerName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}}
				} else if _, ok := namespaces[namespaceName].topLevels[podOwnersKind[podName]+"__"+controllerName].containers[containerName]; !ok {
					namespaces[namespaceName].topLevels[podOwnersKind[podName]+"__"+controllerName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}}
				}
			}
			if _, ok := namespaces[namespaceName].pointers[podOwnersKind[podName]+"__"+controllerName]; !ok {
				namespaces[namespaceName].pointers[podOwnersKind[podName]+"__"+controllerName] = namespaces[namespaceName].topLevels[ownerKind+"__"+currentOwner]
			}
		} else {
			currentOwner = podName
			ownerKind = "Pod"
			//Create pod as top owner and add container
			if _, ok := namespaces[namespaceName].topLevels["Pod__"+podName]; ok {
			} else {
				namespaces[namespaceName].topLevels["Pod__"+podName] = &topLevel{name: podName, kind: "Pod", containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
			}
			namespaces[namespaceName].topLevels["Pod__"+podName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}}
		}
		if _, ok := namespaces[namespaceName].pointers[podName]; !ok {
			namespaces[namespaceName].pointers["Pod__"+podName] = namespaces[namespaceName].topLevels[ownerKind+"__"+currentOwner]
		}
		namespaces[namespaceName].topLevels[ownerKind+"__"+currentOwner].namespace = namespaceName
	}

	//for printing containers
	/*
		for i := range namespaces {
			fmt.Println("\n\nnamespace: " + i)
			for j, v := range namespaces[i].topLevels {
				fmt.Println("\n\n  owner name: " + j + "\n  owner kind: " + v.kind + "\n  namespace: " + i + "\n  containers:")
				for k := range namespaces[i].topLevels[j].containers {
					fmt.Println("  - " + k)
				}
			}
		} */

	var query = `sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "cpuLimit")

	//Get the CPU Request for container
	query = `sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "cpuRequest")

	query = `container_spec_cpu_shares{name!~"k8s_POD_.*"}`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetricString(result, "namespace", "pod_name", "container_name", "conLabel")

	query = `kube_pod_container_info`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetricString(result, "namespace", "pod", "container", "conInfo")

	query = `sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "memLimit")

	query = `sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "memRequest")

	query = `sum(kube_pod_container_status_restarts_total) by (pod,namespace,container)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "restarts")

	query = `sum(kube_pod_container_status_terminated) by (pod,namespace,container)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod", "container", "powerState")

	query = `kube_pod_info`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "pod", "podInfo", "Pod")

	query = `label_replace(kube_pod_labels, "node_instance", "$1", "instance", "(.*)")`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "pod", "podLabel", "Pod")

	query = `kube_pod_created`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "pod", "creationTime", "Pod")

	//Namespace Metrics
	//Get the namespace labels
	query = `kube_namespace_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getNamespaceMetricString(result, "namespace", "namespaceLabel")

	//Get the CPU and Memory Limit and Request quotes for the namespace.
	query = `kube_limitrange`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getNamespacelimits(result, "namespace")

	query = `kube_pod_created`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "pod", "creationTime", "Pod")

	//Get the controller labels
	query = `kube_statefulset_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "statefulset", "label", "StatefulSet")

	query = `kube_job_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "job_name", "label", "Job")

	query = `kube_job_info`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "job_name", "info", "Job")

	query = `kube_daemonset_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "daemonset", "label", "DaemonSet")

	query = `kube_replicaset_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "replicaset", "label", "ReplicaSet")

	query = `kube_deployment_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "deployment", "label", "Deployment")

	query = `kube_cronjob_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "cronjob", "label", "CronJob")

	query = `kube_cronjob_info`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "cronjob", "info", "CronJob")

	//get creation time
	query = `kube_cronjob_created`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "cronjob", "creationTime", "CronJob")

	query = `kube_deployment_created`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "deployment", "creationTime", "Deployment")

	query = `kube_replicaset_created`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "replicaset", "creationTime", "ReplicaSet")

	query = `kube_daemonset_created`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "daemonset", "creationTime", "DaemonSet")

	query = `kube_replicationcontroller_created`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "replicationcontroller", "creationTime", "ReplicationController")

	query = `kube_statefulset_created`                                                     
	result = prometheus.MetricCollect(promaddress, query, start, end)                     
	getMidMetric(result, "namespace", "statefulset", "creationTime", "StatefulSet")        

	query = `kube_job_created`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "job", "creationTime", "Job")

	query = `kube_pod_created`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "replicaset", "creationTime", "ReplicaSet")

	currentSizeWrite, err := os.Create("./data/currentSize.csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(currentSizeWrite, "cluster,namespace,pod,container,Datetime,currentSize\n")

	//Get the current size of the controller will query each of the differnt types of controller
	query = `kube_replicaset_spec_replicas`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "replicaset", "currentSize", "ReplicaSet")
	writeWorkloadMid(currentSizeWrite, result, "namespace", "replicaset", clusterName, promAddr, "ReplicaSet")

	query = `kube_replicationcontroller_spec_replicas`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "replicationcontroller", "currentSize", "ReplicationController")
	writeWorkloadMid(currentSizeWrite, result, "namespace", "replicationcontroller", clusterName, promAddr, "ReplicationController")

	query = `kube_daemonset_status_number_available`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "daemonset", "currentSize", "DaemonSet")
	writeWorkloadMid(currentSizeWrite, result, "namespace", "daemonset", clusterName, promAddr, "DaemonSet")

	query = `kube_statefulset_replicas`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "statefulset", "currentSize", "StatefulSet")
	writeWorkloadMid(currentSizeWrite, result, "namespace", "statefulset", clusterName, promAddr, "StatefulSet")

	query = `kube_job_spec_parallelism`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "job_name", "currentSize", "Job")
	writeWorkloadMid(currentSizeWrite, result, "namespace", "job_name", clusterName, promAddr, "Job")

	query = `sum(max(kube_job_spec_parallelism) by (job) * on (job) group_right kube_job_owner) by (owner_name, namespace)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "owner_name", "currentSize", "CronJob")
	writeWorkloadMid(currentSizeWrite, result, "namespace", "owner_name", clusterName, promAddr, "CronJob")

	query = `sum(max(kube_replicaset_spec_replicas) by (replicaset) * on (replicaset) group_right kube_replicaset_owner) by (owner_name, namespace)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "owner_name", "currentSize", "Deployment")
	writeWorkloadMid(currentSizeWrite, result, "namespace", "owner_name", clusterName, promAddr, "Deployment")

	currentSizeWrite.Close()
	
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

	//for printing label maps
	/*
		for i := range namespaces {
			for j := range namespaces[i].topLevels {
				for k := range namespaces[i].topLevels[j].containers {
					fmt.Println("\n" + k)
					for l, v := range namespaces[i].topLevels[j].containers[k].labelMap {
						fmt.Println("  " + l + " --- " + v)
					}
				}
			}
		}*/
	writeAttributes(clusterName, promAddr)
	writeConfig(clusterName, promAddr)
}
