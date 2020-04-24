//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"fmt"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/logger"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var systems = map[string]*namespace{}
var entityKind = "Container"

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
func Metrics(args *common.Parameters) string {
	//Setup variables used in the code.
	var errors = ""
	var historyInterval time.Duration
	historyInterval = 0
	var query string
	var result model.Value
	var logLine string
	var start, end time.Time
	var rslt model.Matrix

	var podOwners = map[string]string{}
	var podOwnersKind = map[string]string{}
	var replicaSetOwners = map[string]string{}
	var jobOwners = map[string]string{}

	start, end = prometheus.TimeRange(*args.Interval, *args.IntervalSize, *args.CurrentTime, historyInterval)
	range5Min := v1.Range{Start: start, End: end, Step: time.Minute * 5}

	collectArgs := &prometheus.CollectionArgs{
		EntityKind: &entityKind,
		PromURL:    args.PromURL,
		Query:      &query,
		Range:      &range5Min,
	}

	//querys gathering hierarchy information for the containers
	query = `sum(kube_pod_owner{owner_name!="<none>"}) by (namespace, pod, owner_name, owner_kind)`
	result, logLine = prometheus.MetricCollect(collectArgs, "pods", true)
	if logLine == "" {
		rslt = result.(model.Matrix)
		for i := 0; i < rslt.Len(); i++ {
			podOwners[string(rslt[i].Metric["pod"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_name"])
			podOwnersKind[string(rslt[i].Metric["pod"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_kind"])
		}
	} else {
		return errors + logLine
	}

	query = `sum(kube_replicaset_owner{owner_name!="<none>"}) by (namespace, replicaset, owner_name)`
	result, logLine = prometheus.MetricCollect(collectArgs, "replicasets", false)
	if logLine == "" {
		rslt = result.(model.Matrix)
		for i := 0; i < rslt.Len(); i++ {
			replicaSetOwners[string(rslt[i].Metric["replicaset"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_name"])
		}
	}

	query = `sum(kube_job_owner{owner_name!="<none>"}) by (namespace, job_name, owner_name)`
	result, logLine = prometheus.MetricCollect(collectArgs, "jobs", false)
	if logLine == "" {
		rslt = result.(model.Matrix)
		for i := 0; i < rslt.Len(); i++ {
			jobOwners[string(rslt[i].Metric["job_name"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_name"])
		}
	}

	query = `max(kube_pod_container_info) by (container, pod, namespace)`
	result, logLine = prometheus.MetricCollect(collectArgs, "containers", true)
	if logLine == "" {
		rslt = result.(model.Matrix)
	} else {
		return errors + logLine
	}

	var currentOwner string

	//Add containers and top owners to structure
	for i := 0; i < rslt.Len(); i++ {

		containerName := string(rslt[i].Metric["container"])
		podName := string(rslt[i].Metric["pod"])
		var ownerKind string

		namespaceName := string(rslt[i].Metric["namespace"])
		if _, ok := systems[namespaceName]; !ok {
			systems[namespaceName] = &namespace{pointers: map[string]*midLevel{}, midLevels: map[string]*midLevel{}, cpuRequest: -1, cpuLimit: -1, memRequest: -1, memLimit: -1, labelMap: map[string]string{}}
		}

		//systems[namespaceName].pods[podName] = &pod{labelMap: map[string]string{}}
		if controllerName, ok := podOwners[podName+"__"+namespaceName]; ok {
			if deploymentName, ok := replicaSetOwners[controllerName+"__"+namespaceName]; ok && podOwnersKind[podName+"__"+namespaceName] == "ReplicaSet" {
				currentOwner = deploymentName
				ownerKind = "Deployment"
				//Create deployment as top owner and add container
				if _, ok := systems[namespaceName].midLevels[ownerKind+"__"+deploymentName]; !ok {
					systems[namespaceName].midLevels[ownerKind+"__"+deploymentName] = &midLevel{name: deploymentName, kind: "Deployment", containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
					systems[namespaceName].midLevels[ownerKind+"__"+deploymentName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, memory: -1, restarts: -1, powerState: 1}
				} else if _, ok := systems[namespaceName].midLevels[ownerKind+"__"+deploymentName].containers[containerName]; !ok {
					systems[namespaceName].midLevels[ownerKind+"__"+deploymentName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, memory: -1, restarts: -1, powerState: 1}
				}
				if _, ok := systems[namespaceName].pointers[ownerKind+"__"+deploymentName]; !ok {
					systems[namespaceName].pointers[ownerKind+"__"+deploymentName] = systems[namespaceName].midLevels[ownerKind+"__"+currentOwner]
				}
			} else if cronJobName, ok := jobOwners[controllerName+"__"+namespaceName]; ok && podOwnersKind[podName+"__"+namespaceName] == "Job" {
				currentOwner = cronJobName
				ownerKind = "CronJob"
				//Create deployment as top owner and add container
				if _, ok := systems[namespaceName].midLevels[ownerKind+"__"+cronJobName]; !ok {
					systems[namespaceName].midLevels[ownerKind+"__"+cronJobName] = &midLevel{name: cronJobName, kind: "CronJob", containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
					systems[namespaceName].midLevels[ownerKind+"__"+cronJobName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, memory: -1, restarts: -1, powerState: 1}
				} else if _, ok := systems[namespaceName].midLevels[ownerKind+"__"+cronJobName].containers[containerName]; !ok {
					systems[namespaceName].midLevels[ownerKind+"__"+cronJobName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, memory: -1, restarts: -1, powerState: 1}
				}
				if _, ok := systems[namespaceName].pointers[ownerKind+"__"+cronJobName]; !ok {
					systems[namespaceName].pointers[ownerKind+"__"+cronJobName] = systems[namespaceName].midLevels[ownerKind+"__"+currentOwner]
				}
			} else {
				currentOwner = controllerName
				ownerKind = podOwnersKind[podName+"__"+namespaceName]
				//Create controller as top owner and add container
				if _, ok := systems[namespaceName].midLevels[podOwnersKind[podName+"__"+namespaceName]+"__"+controllerName]; !ok {
					systems[namespaceName].midLevels[podOwnersKind[podName+"__"+namespaceName]+"__"+controllerName] = &midLevel{name: controllerName, kind: podOwnersKind[podName+"__"+namespaceName], containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
					systems[namespaceName].midLevels[podOwnersKind[podName+"__"+namespaceName]+"__"+controllerName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, memory: -1, restarts: -1, powerState: 1}
				} else if _, ok := systems[namespaceName].midLevels[podOwnersKind[podName+"__"+namespaceName]+"__"+controllerName].containers[containerName]; !ok {
					systems[namespaceName].midLevels[podOwnersKind[podName+"__"+namespaceName]+"__"+controllerName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, memory: -1, restarts: -1, powerState: 1}
				}
			}
			if _, ok := systems[namespaceName].pointers[podOwnersKind[podName+"__"+namespaceName]+"__"+controllerName]; !ok {
				systems[namespaceName].pointers[podOwnersKind[podName+"__"+namespaceName]+"__"+controllerName] = systems[namespaceName].midLevels[ownerKind+"__"+currentOwner]
			}
		} else {
			currentOwner = podName
			ownerKind = "Pod"
			//Create pod as top owner and add container
			if _, ok := systems[namespaceName].midLevels[ownerKind+"__"+podName]; !ok {
				systems[namespaceName].midLevels[ownerKind+"__"+podName] = &midLevel{name: podName, kind: "Pod", containers: map[string]*container{}, labelMap: map[string]string{}, currentSize: -1}
			}
			systems[namespaceName].midLevels[ownerKind+"__"+podName].containers[containerName] = &container{name: containerName, labelMap: map[string]string{}, cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, memory: -1, restarts: -1, powerState: 1}
		}
		if _, ok := systems[namespaceName].pointers["Pod__"+podName]; !ok {
			systems[namespaceName].pointers["Pod__"+podName] = systems[namespaceName].midLevels[ownerKind+"__"+currentOwner]
		}
	}

	//for printing containers
	tempString := ""
	if args.Debug {
		for i := range systems {
			fmt.Println("\nnamespace: " + i)
			tempString += "\nnamespace: " + i
			for j, v := range systems[i].midLevels {
				fmt.Println("- entity name: " + v.name + "\n  entity kind: " + v.kind + "\n  namespace: " + i + "\n  containers:")
				tempString += "- entity name: " + v.name + "\n  entity kind: " + v.kind + "\n  namespace: " + i + "\n  containers:"
				for k := range systems[i].midLevels[j].containers {
					fmt.Println("  - " + k)
					tempString += "  - " + k
				}
			}
		}
		errors += logger.LogError(map[string]string{"message": (tempString)}, "DEBUG")
	}

	//Container metrics
	query = `container_spec_memory_limit_bytes{name!~"k8s_POD_.*"}/1024/1024`
	result, logLine = prometheus.MetricCollect(collectArgs, "memory", false)
	if logLine == "" {
		if args.LabelSuffix == "" && getContainerMetric(result, "namespace", "pod", "container", "memory") {
			//Don't do anything
		} else if getContainerMetric(result, "namespace", "pod_name", "container_name", "memory") {
			args.LabelSuffix = "_name"
		}
	} else {
		errors += logLine
	}

	query = `sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000`
	result, logLine = prometheus.MetricCollect(collectArgs, "cpuLimit", false)
	if logLine == "" {
		getContainerMetric(result, "namespace", "pod", "container", "cpuLimit")
	} else {
		errors += logLine
	}

	query = `sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000`
	result, logLine = prometheus.MetricCollect(collectArgs, "cpuRequest", false)
	if logLine == "" {
		getContainerMetric(result, "namespace", "pod", "container", "cpuRequest")
	} else {
		errors += logLine
	}

	query = `sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024`
	result, logLine = prometheus.MetricCollect(collectArgs, "memLimit", false)
	if logLine == "" {
		getContainerMetric(result, "namespace", "pod", "container", "memLimit")
	} else {
		errors += logLine
	}

	query = `sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024`
	result, logLine = prometheus.MetricCollect(collectArgs, "memRequest", false)
	if logLine == "" {
		getContainerMetric(result, "namespace", "pod", "container", "memRequest")
	} else {
		errors += logLine
	}

	query = `container_spec_cpu_shares{name!~"k8s_POD_.*"}`
	result, logLine = prometheus.MetricCollect(collectArgs, "conLabel", false)
	if logLine == "" {
		getContainerMetricString(result, "namespace", model.LabelName("pod"+args.LabelSuffix), model.LabelName("container"+args.LabelSuffix))
	} else {
		errors += logLine
	}

	query = `kube_pod_container_info`
	result, logLine = prometheus.MetricCollect(collectArgs, "conInfo", false)
	if logLine == "" {
		getContainerMetricString(result, "namespace", "pod", "container")
	} else {
		errors += logLine
	}

	//Pod metrics
	query = `kube_pod_info`
	result, logLine = prometheus.MetricCollect(collectArgs, "podInfo", false)
	if logLine == "" {
		getMidMetricString(result, "namespace", "pod", "Pod")
	} else {
		errors += logLine
	}

	query = `kube_pod_labels`
	result, logLine = prometheus.MetricCollect(collectArgs, "podLabels", false)
	if logLine == "" {
		getMidMetricString(result, "namespace", "pod", "Pod")
	} else {
		errors += logLine
	}

	query = `sum(kube_pod_container_status_restarts_total) by (pod,namespace,container)`
	result, logLine = prometheus.MetricCollect(collectArgs, "restarts", false)
	if logLine == "" {
		getContainerMetric(result, "namespace", "pod", "container", "restarts")
	} else {
		errors += logLine
	}

	query = `sum(kube_pod_container_status_terminated) by (pod,namespace,container)`
	result, logLine = prometheus.MetricCollect(collectArgs, "powerState", false)
	if logLine == "" {
		getContainerMetric(result, "namespace", "pod", "container", "powerState")
	} else {
		errors += logLine
	}

	query = `kube_pod_created`
	result, logLine = prometheus.MetricCollect(collectArgs, "podCreationTime", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "pod", "creationTime", "Pod")
	} else {
		errors += logLine
	}

	//Namespace metrics
	query = `kube_namespace_labels`
	result, logLine = prometheus.MetricCollect(collectArgs, "namespaceLabels", false)
	if logLine == "" {
		getNamespaceMetricString(result, "namespace")
	} else {
		errors += logLine
	}

	query = `kube_namespace_annotations`
	result, logLine = prometheus.MetricCollect(collectArgs, "namespaceAnnotations", false)
	if logLine == "" {
		getNamespaceMetricString(result, "namespace")
	} else {
		errors += logLine
	}

	query = `kube_limitrange`
	result, logLine = prometheus.MetricCollect(collectArgs, "nameSpaceLimitrange", false)
	if logLine == "" {
		getNamespacelimits(result, "namespace")
	} else {
		errors += logLine
	}

	//Deployment metrics
	query = `kube_deployment_labels`
	result, logLine = prometheus.MetricCollect(collectArgs, "labels", false)
	if logLine == "" {
		getMidMetricString(result, "namespace", "deployment", "Deployment")
	} else {
		errors += logLine
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_surge`
	result, logLine = prometheus.MetricCollect(collectArgs, "maxSurge", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "deployment", "maxSurge", "Deployment")
	} else {
		errors += logLine
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_unavailable`
	result, logLine = prometheus.MetricCollect(collectArgs, "maxUnavailable", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "deployment", "maxUnavailable", "Deployment")
	} else {
		errors += logLine
	}

	query = `kube_deployment_metadata_generation`
	result, logLine = prometheus.MetricCollect(collectArgs, "metadataGeneration", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "deployment", "metadataGeneration", "Deployment")
	} else {
		errors += logLine
	}

	query = `kube_deployment_created`
	result, logLine = prometheus.MetricCollect(collectArgs, "deploymentCreated", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "deployment", "creationTime", "Deployment")
	} else {
		errors += logLine
	}

	//ReplicaSet metrics
	query = `kube_replicaset_labels`
	result, logLine = prometheus.MetricCollect(collectArgs, "replicaSetLabels", false)
	if logLine == "" {
		getMidMetricString(result, "namespace", "replicaset", "ReplicaSet")
	} else {
		errors += logLine
	}

	query = `kube_replicaset_created`
	result, logLine = prometheus.MetricCollect(collectArgs, "replicaSetCreated", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "replicaset", "creationTime", "ReplicaSet")
	} else {
		errors += logLine
	}

	//ReplicationController metrics
	query = `kube_replicationcontroller_created`
	result, logLine = prometheus.MetricCollect(collectArgs, "replicationControllerCreated", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "replicationcontroller", "creationTime", "ReplicationController")
	} else {
		errors += logLine
	}

	//DaemonSet metrics
	query = `kube_daemonset_labels`
	result, logLine = prometheus.MetricCollect(collectArgs, "daemonSetLabels", false)
	if logLine == "" {
		getMidMetricString(result, "namespace", "daemonset", "DaemonSet")
	} else {
		errors += logLine
	}

	query = `kube_daemonset_created`
	result, logLine = prometheus.MetricCollect(collectArgs, "daemonSetCreated", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "daemonset", "creationTime", "DaemonSet")
	} else {
		errors += logLine
	}

	//StatefulSet metrics
	query = `kube_statefulset_labels`
	result, logLine = prometheus.MetricCollect(collectArgs, "statefulSetLabels", false)
	if logLine == "" {
		getMidMetricString(result, "namespace", "statefulset", "StatefulSet")
	} else {
		errors += logLine
	}

	query = `kube_statefulset_created`
	result, logLine = prometheus.MetricCollect(collectArgs, "statefulSetCreated", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "statefulset", "creationTime", "StatefulSet")
	} else {
		errors += logLine
	}

	//Job metrics
	query = `kube_job_info * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, logLine = prometheus.MetricCollect(collectArgs, "jobInfo", false)
	if logLine == "" {
		getMidMetricString(result, "namespace", "job_name", "Job")
	} else {
		errors += logLine
	}

	query = `kube_job_labels * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, logLine = prometheus.MetricCollect(collectArgs, "jobLabel", false)
	if logLine == "" {
		getMidMetricString(result, "namespace", "job_name", "Job")
	} else {
		errors += logLine
	}

	query = `kube_job_spec_completions * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, logLine = prometheus.MetricCollect(collectArgs, "jobSpecCompletions", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "job_name", "specCompletions", "Job")
	} else {
		errors += logLine
	}

	query = `kube_job_spec_parallelism * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, logLine = prometheus.MetricCollect(collectArgs, "jobSpecParallelism", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "job_name", "specParallelism", "Job")
	} else {
		errors += logLine
	}

	query = `kube_job_status_completion_time * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, logLine = prometheus.MetricCollect(collectArgs, "jobStatusCompletionTime", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "job_name", "statusCompletionTime", "Job")
	} else {
		errors += logLine
	}

	query = `kube_job_status_start_time * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, logLine = prometheus.MetricCollect(collectArgs, "jobStatusStartTime", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "job_name", "statusStartTime", "Job")
	} else {
		errors += logLine
	}

	query = `kube_job_created`
	result, logLine = prometheus.MetricCollect(collectArgs, "jobCreated", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "job", "creationTime", "Job")
	} else {
		errors += logLine
	}

	//CronJob metrics
	query = `kube_cronjob_labels`
	result, logLine = prometheus.MetricCollect(collectArgs, "cronJobLabels", false)
	if logLine == "" {
		getMidMetricString(result, "namespace", "cronjob", "CronJob")
	} else {
		errors += logLine
	}

	query = `kube_cronjob_info`
	result, logLine = prometheus.MetricCollect(collectArgs, "cronJobInfo", false)
	if logLine == "" {
		getMidMetricString(result, "namespace", "cronjob", "CronJob")
	} else {
		errors += logLine
	}

	query = `kube_cronjob_next_schedule_time`
	result, logLine = prometheus.MetricCollect(collectArgs, "cronJobNextScheduleTime", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "cronjob", "nextScheduleTime", "CronJob")
	} else {
		errors += logLine
	}

	query = `kube_cronjob_status_last_schedule_time`
	result, logLine = prometheus.MetricCollect(collectArgs, "cronJobStatusLastScheduleTime", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "cronjob", "lastScheduleTime", "CronJob")
	} else {
		errors += logLine
	}

	query = `kube_cronjob_status_active`
	result, logLine = prometheus.MetricCollect(collectArgs, "cronJobStatusActive", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "cronjob", "statusActive", "CronJob")
	} else {
		errors += logLine
	}

	query = `kube_cronjob_created`
	result, logLine = prometheus.MetricCollect(collectArgs, "cronJobCreated", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "cronjob", "creationTime", "CronJob")
	} else {
		errors += logLine
	}

	//HPA metrics
	query = `kube_hpa_labels`
	result, logLine = prometheus.MetricCollect(collectArgs, "hpaLabels", false)
	if logLine == "" {
		getHPAMetricString(result, "namespace", "hpa", *args.ClusterName, *args.PromAddress)
	} else {
		errors += logLine
	}

	//Current size workloads
	currentSizeWrite, err := os.Create("./data/container/currentSize.csv")
	if err != nil {
		return errors + logger.LogError(map[string]string{"entity": entityKind, "message": err.Error()}, "ERROR")
	}
	fmt.Fprintf(currentSizeWrite, "cluster,namespace,entity_name,entity_type,container,Datetime,Auto Scaling - In Service Instances\n")

	query = `kube_replicaset_spec_replicas`
	result, logLine = prometheus.MetricCollect(collectArgs, "replicaSetSpecReplicas", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "replicaset", "currentSize", "ReplicaSet")
	} else {
		errors += logLine
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "replicaset", *args.ClusterName, *args.PromAddress, "ReplicaSet")

	query = `kube_replicationcontroller_spec_replicas`
	result, logLine = prometheus.MetricCollect(collectArgs, "replicationcontroller_spec_replicas", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "replicationcontroller", "currentSize", "ReplicationController")
	} else {
		errors += logLine
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "replicationcontroller", *args.ClusterName, *args.PromAddress, "ReplicationController")

	query = `kube_daemonset_status_number_available`
	result, logLine = prometheus.MetricCollect(collectArgs, "daemonSetStatusNumberAvailable", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "daemonset", "currentSize", "DaemonSet")
	} else {
		errors += logLine
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "daemonset", *args.ClusterName, *args.PromAddress, "DaemonSet")

	query = `kube_statefulset_replicas`
	result, logLine = prometheus.MetricCollect(collectArgs, "statefulSetReplicas", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "statefulset", "currentSize", "StatefulSet")
	} else {
		errors += logLine
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "statefulset", *args.ClusterName, *args.PromAddress, "StatefulSet")

	query = `kube_job_spec_parallelism`
	result, logLine = prometheus.MetricCollect(collectArgs, "jobSpecParallelism", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "job_name", "currentSize", "Job")
	} else {
		errors += logLine
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "job_name", *args.ClusterName, *args.PromAddress, "Job")

	query = `sum(max(kube_job_spec_parallelism) by (namespace,job_name) * on (namespace,job_name) group_right max(kube_job_owner) by (namespace, job_name, owner_name)) by (owner_name, namespace)`
	result, logLine = prometheus.MetricCollect(collectArgs, "cronJobSpecParallelism", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "owner_name", "currentSize", "CronJob")
	} else {
		errors += logLine
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "owner_name", *args.ClusterName, *args.PromAddress, "CronJob")

	query = `sum(max(kube_replicaset_spec_replicas) by (namespace,replicaset) * on (namespace,replicaset) group_right max(kube_replicaset_owner) by (namespace, replicaset, owner_name)) by (owner_name, namespace)`
	result, logLine = prometheus.MetricCollect(collectArgs, "replicaSetSpecReplicas", false)
	if logLine == "" {
		getMidMetric(result, "namespace", "owner_name", "currentSize", "Deployment")
	} else {
		errors += logLine
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "owner_name", *args.ClusterName, *args.PromAddress, "Deployment")

	currentSizeWrite.Close()

	errors += writeAttributes(*args.ClusterName, *args.PromAddress)
	errors += writeConfig(*args.ClusterName, *args.PromAddress)

	queryPrefix := ``
	querySuffix := ``
	if args.LabelSuffix != "" {
		queryPrefix = `label_replace(`
		querySuffix = `, "pod", "$1", "pod_name", "(.*)")`
	}

	//Container workloads
	query = queryPrefix + `round(sum(irate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)*1000,1)` + querySuffix
	errors += getWorkload("cpu_mCores_workload", "CPU Utilization in mCores", query, "max", args)
	errors += getWorkload("cpu_mCores_workload", "Prometheus CPU Utilization in mCores", query, "avg", args)

	query = queryPrefix + `sum(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)` + querySuffix
	errors += getWorkload("mem_workload", "Raw Mem Utilization", query, "max", args)
	errors += getWorkload("mem_workload", "Prometheus Raw Mem Utilization", query, "avg", args)

	query = queryPrefix + `sum(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)` + querySuffix
	errors += getWorkload("rss_workload", "Actual Memory Utilization", query, "max", args)
	errors += getWorkload("rss_workload", "Prometheus Actual Memory Utilization", query, "avg", args)

	query = queryPrefix + `sum(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)` + querySuffix
	errors += getWorkload("disk_workload", "Raw Disk Utilization", query, "max", args)
	errors += getWorkload("disk_workload", "Prometheus Raw Disk Utilization", query, "avg", args)

	/*
		query = `label_replace(sum(irate(container_fs_read_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name), "pod", "$1", "pod_name", "(.*)")`
		errors += getWorkload(promaddress, "fs_read_seconds_workload", "FS Read Seconds", query, "max", *args.ClusterName, *args.PromAddress, *args.Interval, *args.IntervalSize, *args.History, currentTime)
		errors += getWorkload(promaddress, "fs_read_seconds_workload", "FS Read Seconds", query, "avg", *args.ClusterName, *args.PromAddress, *args.Interval, *args.IntervalSize, *args.History, currentTime)

		query = `label_replace(sum(irate(container_fs_write_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name), "pod", "$1", "pod_name", "(.*)")`
		errors += getWorkload(promaddress, "fs_write_seconds_workload", "FS Write Seconds", query, "max", *args.ClusterName, *args.PromAddress, *args.Interval, *args.IntervalSize, *args.History, currentTime)
		errors += getWorkload(promaddress, "fs_write_seconds_workload", "FS Write Seconds", query, "avg", *args.ClusterName, *args.PromAddress, *args.Interval, *args.IntervalSize, *args.History, currentTime)

		query = `label_replace(sum(irate(container_fs_io_time_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod_name,namespace,container_name), "pod", "$1", "pod_name", "(.*)")`
		errors += getWorkload(promaddress, "fs_time_seconds_workload", "FS Time Seconds", query, "max", *args.ClusterName, *args.PromAddress, *args.Interval, *args.IntervalSize, *args.History, currentTime)
		errors += getWorkload(promaddress, "fs_time_seconds_workload", "FS Time Seconds", query, "avg", *args.ClusterName, *args.PromAddress, *args.Interval, *args.IntervalSize, *args.History, currentTime)
	*/

	if args.LabelSuffix != "" {
		queryPrefix = `label_replace(`
		querySuffix = `, "container_name", "$1", "container", "(.*)")`
	}
	query = queryPrefix + `sum(irate(kube_pod_container_status_restarts_total{name!~"k8s_POD_.*"}[1h])) by (instance,pod,namespace,container)` + querySuffix
	errors += getWorkload("restarts", "Restarts", query, "max", args)

	/*
		//Deployment workloads
		query = `kube_deployment_status_replicas_available`
		errors += getDeploymentWorkload(promaddress, "status_replicas_available", "Status Replicas Available", query, *args.ClusterName, *args.PromAddress, *args.Interval, *args.IntervalSize, *args.History, currentTime)

		query = `kube_deployment_status_replicas`
		errors += getDeploymentWorkload(promaddress, "status_replicas", "Status Replicas", query, *args.ClusterName, *args.PromAddress, *args.Interval, *args.IntervalSize, *args.History, currentTime)

		query = `kube_deployment_spec_replicas`
		errors += getDeploymentWorkload(promaddress, "spec_replicas", "Spec Replicas", query, *args.ClusterName, *args.PromAddress, *args.Interval, *args.IntervalSize, *args.History, currentTime)
	*/

	if args.LabelSuffix == "" {
		query = `kube_hpa_status_condition{status="ScalingLimited",condition="true"}`
	} else {
		query = `kube_hpa_status_condition{status="true",condition="ScalingLimited"}`
	}
	errors += getHPAWorkload("condition_scaling_limited", "Scaling Limited", query, args)

	//HPA workloads
	query = `kube_hpa_spec_max_replicas`
	errors += getHPAWorkload("max_replicas", "Auto Scaling - Maximum Size", query, args)

	query = `kube_hpa_spec_min_replicas`
	errors += getHPAWorkload("min_replicas", "Auto Scaling - Minimum Size", query, args)

	query = `kube_hpa_status_current_replicas`
	errors += getHPAWorkload("current_replicas", "Auto Scaling - Total Instances", query, args)

	/*
		query = `kube_hpa_status_desired_replicas`
		errors += getHPAWorkload(promaddress, "desired_replicas", "Desired Replicas", query, *args.ClusterName, *args.PromAddress, *args.Interval, *args.IntervalSize, *args.History, currentTime)
	*/

	return errors
}
