//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

var systems = map[string]*namespace{}

type namespace struct {
	pointers                                   map[string]*midLevel
	midLevels                                  map[string]*midLevel
	cpuLimit, cpuRequest, memLimit, memRequest int
	labelMap                                   map[string]string
}

//midLevel is used to hold information related to the highest owner of any containers
type midLevel struct {
	name, kind            string
	containers            map[string]*container
	currentSize, restarts int
	creationTime          int64
	labelMap              map[string]string
}

//container is used to hold information related to containers
type container struct {
	memory, cpuLimit, cpuRequest, memLimit, memRequest, restarts, powerState int
	name                                                                     string
	labelMap                                                                 map[string]string
}

//Metrics function to collect data related to containers.
func Metrics(clusterName, promProtocol, promAddr, promPort, interval string, intervalSize, history int, debug bool, currentTime time.Time) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var query, promaddress string
	var result model.Value
	var start, end time.Time
	var rslt model.Matrix

	var podNamespaces = map[string]string{}
	var podOwners = map[string]string{}
	var podOwnersKind = map[string]string{}
	var replicaSetOwners = map[string]string{}
	var jobOwners = map[string]string{}

	start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
	promaddress = promProtocol + "://" + promAddr + ":" + promPort

	//querys gathering hierarchy information for the containers
	query = `kube_pod_owner{owner_name!="<none>"}`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	rslt = result.(model.Matrix)
	for i := 0; i < rslt.Len(); i++ {
		podOwners[string(rslt[i].Metric["pod"])] = string(rslt[i].Metric["owner_name"])
		podOwnersKind[string(rslt[i].Metric["pod"])] = string(rslt[i].Metric["owner_kind"])
	}

	query = `kube_replicaset_owner{owner_name!="<none>"}`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	rslt = result.(model.Matrix)
	for i := 0; i < rslt.Len(); i++ {
		replicaSetOwners[string(rslt[i].Metric["replicaset"])] = string(rslt[i].Metric["owner_name"])
	}

	query = `kube_job_owner{owner_name!="<none>"}`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	rslt = result.(model.Matrix)
	for i := 0; i < rslt.Len(); i++ {
		jobOwners[string(rslt[i].Metric["job_name"])] = string(rslt[i].Metric["owner_name"])
	}

	query = `max(kube_pod_container_info) by (container, pod, namespace)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	rslt = result.(model.Matrix)
	for i := 0; i < rslt.Len(); i++ {
		podNamespaces[string(rslt[i].Metric["pod"])] = string(rslt[i].Metric["namespace"])
	}

	var currentOwner string

	//Add containers and top owners to structure
	for i := 0; i < rslt.Len(); i++ {

		containerName := string(rslt[i].Metric["container"])
		podName := string(rslt[i].Metric["pod"])
		var ownerKind string

		namespaceName := podNamespaces[podName]
		if _, ok := systems[namespaceName]; !ok {
			systems[namespaceName] = &namespace{pointers: map[string]*midLevel{}, midLevels: map[string]*midLevel{}, cpuRequest: -1, cpuLimit: -1, memRequest: -1, memLimit: -1, labelMap: map[string]string{}}
		}

		//systems[namespaceName].pods[podName] = &pod{labelMap: map[string]string{}}
		if controllerName, ok := podOwners[podName]; ok {
			if deploymentName, ok := replicaSetOwners[controllerName]; ok && podOwnersKind[podName] == "ReplicaSet" {
				currentOwner = deploymentName
				ownerKind = "Deployment"
				//Create deployment as top owner and add container
				if _, ok := systems[namespaceName].midLevels[ownerKind+"__"+deploymentName]; !ok {
					systems[namespaceName].midLevels[ownerKind+"__"+deploymentName] = &midLevel{name: deploymentName, kind: "Deployment", containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
					systems[namespaceName].midLevels[ownerKind+"__"+deploymentName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1}
				} else if _, ok := systems[namespaceName].midLevels[ownerKind+"__"+deploymentName].containers[containerName]; !ok {
					systems[namespaceName].midLevels[ownerKind+"__"+deploymentName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1}
				}
				if _, ok := systems[namespaceName].pointers[ownerKind+"__"+deploymentName]; !ok {
					systems[namespaceName].pointers[ownerKind+"__"+deploymentName] = systems[namespaceName].midLevels[ownerKind+"__"+currentOwner]
				}
			} else if cronJobName, ok := jobOwners[controllerName]; ok && podOwnersKind[podName] == "Job" {
				currentOwner = cronJobName
				ownerKind = "CronJob"
				//Create deployment as top owner and add container
				if _, ok := systems[namespaceName].midLevels[ownerKind+"__"+cronJobName]; !ok {
					systems[namespaceName].midLevels[ownerKind+"__"+cronJobName] = &midLevel{name: cronJobName, kind: "CronJob", containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
					systems[namespaceName].midLevels[ownerKind+"__"+cronJobName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1}
				} else if _, ok := systems[namespaceName].midLevels[ownerKind+"__"+cronJobName].containers[containerName]; !ok {
					systems[namespaceName].midLevels[ownerKind+"__"+cronJobName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1}
				}
				if _, ok := systems[namespaceName].pointers[ownerKind+"__"+cronJobName]; !ok {
					systems[namespaceName].pointers[ownerKind+"__"+cronJobName] = systems[namespaceName].midLevels[ownerKind+"__"+currentOwner]
				}
			} else {
				currentOwner = controllerName
				ownerKind = podOwnersKind[podName]
				//Create controller as top owner and add container
				if _, ok := systems[namespaceName].midLevels[podOwnersKind[podName]+"__"+controllerName]; !ok {
					systems[namespaceName].midLevels[podOwnersKind[podName]+"__"+controllerName] = &midLevel{name: controllerName, kind: podOwnersKind[podName], containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
					systems[namespaceName].midLevels[podOwnersKind[podName]+"__"+controllerName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1}
				} else if _, ok := systems[namespaceName].midLevels[podOwnersKind[podName]+"__"+controllerName].containers[containerName]; !ok {
					systems[namespaceName].midLevels[podOwnersKind[podName]+"__"+controllerName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1}
				}
			}
			if _, ok := systems[namespaceName].pointers[podOwnersKind[podName]+"__"+controllerName]; !ok {
				systems[namespaceName].pointers[podOwnersKind[podName]+"__"+controllerName] = systems[namespaceName].midLevels[ownerKind+"__"+currentOwner]
			}
		} else {
			currentOwner = podName
			ownerKind = "Pod"
			//Create pod as top owner and add container
			if _, ok := systems[namespaceName].midLevels[ownerKind+"__"+podName]; !ok {
				systems[namespaceName].midLevels[ownerKind+"__"+podName] = &midLevel{name: podName, kind: "Pod", containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
			}
			systems[namespaceName].midLevels[ownerKind+"__"+podName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1}
		}
		if _, ok := systems[namespaceName].pointers["Pod__"+podName]; !ok {
			systems[namespaceName].pointers["Pod__"+podName] = systems[namespaceName].midLevels[ownerKind+"__"+currentOwner]
		}
	}

	//for printing containers
	/*
		for i, vi := range systems {
			fmt.Println("\n\nnamespace: " + i)
			for j, v := range systems[i].midLevels {
				fmt.Println("\n\n  owner name: " + j + "\n  owner kind: " + v.kind + "\n  namespace: " + i + "\n  containers:")
				for k := range systems[i].midLevels[j].containers {
					fmt.Println("  - " + k)
				}
			}
		} */

	query = `container_spec_memory_limit_bytes{name!~"k8s_POD_.*"}/1024/1024`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getContainerMetric(result, "namespace", "pod_name", "container_name", "memory")

	query = `sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000`
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

	query = `kube_pod_labels`
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

	//Deployment metrics
	query = `kube_deployment_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "deployment", "deploymentLabel", "Deployment")

	query = `kube_deployment_spec_strategy_rollingupdate_max_surge`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "deployment", "maxSurge", "Deployment")

	query = `kube_deployment_spec_strategy_rollingupdate_max_unavailable`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "deployment", "maxUnavailable", "Deployment")

	query = `kube_deployment_metadata_generation`
	getMidMetric(result, "namespace", "deployment", "metadataGeneration", "Deployment")

	//fmt.Println(currentTime)
	query = `kube_deployment_status_replicas_available`
	getDeploymentWorkload(promaddress, "status_replicas_available", "Status Replicas Available", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `kube_deployment_status_replicas`
	getDeploymentWorkload(promaddress, "status_replicas", "Status Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `kube_deployment_spec_replicas`
	getDeploymentWorkload(promaddress, "spec_replicas", "Spec Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	//CronJob & Job metrics
	query = `kube_cronjob_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "cronjob", "cronjobLabel", "CronJob")

	query = `kube_cronjob_info`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "cronjob", "cronjobInfo", "CronJob")

	query = `kube_cronjob_next_schedule_time`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "cronjob", "nextScheduleTime", "CronJob")

	query = `kube_cronjob_status_last_schedule_time`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "cronjob", "lastScheduleTime", "CronJob")

	query = `kube_cronjob_status_active`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "cronjob", "statusActive", "CronJob")

	query = `kube_job_info * on (namespace,job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "job_name", "jobInfo", "Job")

	query = `kube_job_labels * on (namespace,job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetricString(result, "namespace", "job_name", "jobLabel", "Job")

	query = `kube_job_spec_completions * on (namespace,job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "job_name", "specCompletions", "Job")

	query = `kube_job_spec_parallelism * on (namespace,job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "job_name", "specParallelism", "Job")

	query = `kube_job_status_completion_time * on (namespace,job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "job_name", "statusCompletionTime", "Job")

	query = `kube_job_status_start_time * on (namespace,job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "job_name", "statusStartTime", "Job")
	/*
		query = `kube_job_status_active * on (namespace,job_name) group_left (owner_name) kube_job_owner`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getMidMetric(result, "namespace", "job", "statusActive", "Job")

		query = `kube_job_status_failed * on (namespace,job_name) group_left (owner_name) kube_job_owner`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getMidMetric(result, "namespace", "job", "statusFailed", "Job")

		query = `kube_job_status_succeeded * on (namespace,job_name) group_left (owner_name) kube_job_owner`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getMidMetric(result, "namespace", "job", "statusSucceeded", "Job")

		query = `kube_job_complete * on (namespace,job_name) group_left (owner_name) kube_job_owner`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getMidMetric(result, "namespace", "job", "complete", "Job")*/

	currentSizeWrite, err := os.Create("./data/container/currentSize.csv")
	if err != nil {
		log.Println(err)
	}
	fmt.Fprintf(currentSizeWrite, "cluster,namespace,entity_name,entity_type,container,Datetime,currentSize\n")

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

	query = `sum(max(kube_job_spec_parallelism) by (job) * on (namespace,job) group_right kube_job_owner) by (owner_name, namespace)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "owner_name", "currentSize", "CronJob")
	writeWorkloadMid(currentSizeWrite, result, "namespace", "owner_name", clusterName, promAddr, "CronJob")

	query = `sum(max(kube_replicaset_spec_replicas) by (replicaset) * on (namespace,replicaset) group_right kube_replicaset_owner) by (owner_name, namespace)`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getMidMetric(result, "namespace", "owner_name", "currentSize", "Deployment")
	writeWorkloadMid(currentSizeWrite, result, "namespace", "owner_name", clusterName, promAddr, "Deployment")

	currentSizeWrite.Close()

	query = `label_replace(round(sum(rate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)*1000,1), "pod", "$1", "pod_name", "(.*)")`
	getWorkload(promaddress, "cpu_mCores_workload", "CPU Utilization in mCores", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `label_replace(sum(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind), "pod", "$1", "pod_name", "(.*)")`
	getWorkload(promaddress, "mem_workload", "Raw Mem Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `label_replace(sum(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind), "pod", "$1", "pod_name", "(.*)")`
	getWorkload(promaddress, "rss_workload", "Actual Memory Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `label_replace(sum(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind), "pod", "$1", "pod_name", "(.*)")`
	getWorkload(promaddress, "disk_workload", "Raw Disk Utilization", query, "max", clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `label_replace(round(sum(rate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name,owner_name,owner_kind)*1000,1), "pod", "$1", "pod_name", "(.*)")`
	getWorkload(promaddress, "cpu_mCores_workload", "CPU Utilization in mCores", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `label_replace(sum(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind), "pod", "$1", "pod_name", "(.*)")`
	getWorkload(promaddress, "mem_workload", "Raw Mem Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `label_replace(sum(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind), "pod", "$1", "pod_name", "(.*)")`
	getWorkload(promaddress, "rss_workload", "Actual Memory Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)
	query = `label_replace(sum(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod_name,namespace,container_name,owner_name,owner_kind), "pod", "$1", "pod_name", "(.*)")`
	getWorkload(promaddress, "disk_workload", "Raw Disk Utilization", query, "avg", clusterName, promAddr, interval, intervalSize, history, currentTime)

	//HPA metrics
	query = `kube_hpa_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getHPAMetricString(result, "namespace", "hpa", "hpaLabel", clusterName, promAddr)

	query = `kube_hpa_spec_max_replicas`
	getHPAWorkload(promaddress, "max_replicas", "Max Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `kube_hpa_spec_min_replicas`
	getHPAWorkload(promaddress, "min_replicas", "Min Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	/*
		query = `kube_hpa_status_condition{status="AbleToScale",condition="true"}`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getHPAMetric(result, "namespace", "hpa", "ableToScale")

		query = `kube_hpa_status_condition{status="ScalingActive",condition="true"}`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getHPAMetric(result, "namespace", "hpa", "scalingActive")
	*/
	query = `kube_hpa_status_condition{status="ScalingLimited",condition="true"}`
	getHPAWorkload(promaddress, "condition_scaling_limited", "Scaling Limited", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `kube_hpa_status_current_replicas`
	getHPAWorkload(promaddress, "current_replicas", "Current Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `kube_hpa_status_desired_replicas`
	getHPAWorkload(promaddress, "desired_replicas", "Desired Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	//for printing label maps
	/*
		for i := range systems {
			for j := range systems[i].midLevels {
				for k := range systems[i].midLevels[j].containers {
					fmt.Println("\n" + k)
					for l, v := range systems[i].midLevels[j].containers[k].labelMap {
						fmt.Println("  " + l + " --- " + v)
					}
				}
			}
		}*/
	writeAttributes(clusterName, promAddr)
	writeConfig(clusterName, promAddr)
}
