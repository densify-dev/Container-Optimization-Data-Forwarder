//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"fmt"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
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
func Metrics(args *common.Parameters) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var query string
	var result model.Value
	var rslt model.Matrix

	var podOwners = map[string]string{}
	var podOwnersKind = map[string]string{}
	var replicaSetOwners = map[string]string{}
	var jobOwners = map[string]string{}

	range5Min := prometheus.TimeRange(args, historyInterval, time.Minute*5)

	collectArgs := &prometheus.CollectionArgs{
		Query: &query,
		Range: &range5Min,
	}

	//querys gathering hierarchy information for the containers
	query = `sum(kube_pod_owner{owner_name!="<none>"}) by (namespace, pod, owner_name, owner_kind)`
	result = prometheus.MetricCollect(args, collectArgs, "pods", true)
	if result != nil {
		rslt = result.(model.Matrix)
		for i := 0; i < rslt.Len(); i++ {
			podOwners[string(rslt[i].Metric["pod"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_name"])
			podOwnersKind[string(rslt[i].Metric["pod"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_kind"])
		}
	} else {
		return
	}

	query = `sum(kube_replicaset_owner{owner_name!="<none>"}) by (namespace, replicaset, owner_name)`
	result = prometheus.MetricCollect(args, collectArgs, "replicasets", false)
	if result != nil {
		rslt = result.(model.Matrix)
		for i := 0; i < rslt.Len(); i++ {
			replicaSetOwners[string(rslt[i].Metric["replicaset"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_name"])
		}
	}

	query = `sum(kube_job_owner{owner_name!="<none>"}) by (namespace, job_name, owner_name)`
	result = prometheus.MetricCollect(args, collectArgs, "jobs", false)
	if result != nil {
		rslt = result.(model.Matrix)
		for i := 0; i < rslt.Len(); i++ {
			jobOwners[string(rslt[i].Metric["job_name"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_name"])
		}
	}

	query = `max(kube_pod_container_info) by (container, pod, namespace)`
	result = prometheus.MetricCollect(args, collectArgs, "containers", true)
	if result != nil {
		rslt = result.(model.Matrix)
	} else {
		return
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
			tempString += "namespace: " + i + "\n"
			for j, v := range systems[i].midLevels {
				tempString += "- entity name: " + v.name + "\n  entity kind: " + v.kind + "\n  namespace: " + i + "\n  containers: \n"
				for k := range systems[i].midLevels[j].containers {
					tempString += "  - " + k + "\n"
				}
			}
		}
		args.DebugLogger.Println("message=Dump of Systesms structure\n" + tempString)
		fmt.Println("message=Dump of Systesms structure\n" + tempString)
	}

	//Container metrics
	query = `container_spec_memory_limit_bytes{name!~"k8s_POD_.*"}/1024/1024`
	result = prometheus.MetricCollect(args, collectArgs, "memory", false)
	if result != nil {
		if args.LabelSuffix == "" && getContainerMetric(result, "namespace", "pod", "container", "memory") {
			//Don't do anything
		} else if getContainerMetric(result, "namespace", "pod_name", "container_name", "memory") {
			args.LabelSuffix = "_name"
		}
	}

	query = `sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000`
	result = prometheus.MetricCollect(args, collectArgs, "cpuLimit", false)
	if result != nil {
		getContainerMetric(result, "namespace", "pod", "container", "cpuLimit")
	}

	query = `sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000`
	result = prometheus.MetricCollect(args, collectArgs, "cpuRequest", false)
	if result != nil {
		getContainerMetric(result, "namespace", "pod", "container", "cpuRequest")
	}

	query = `sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024`
	result = prometheus.MetricCollect(args, collectArgs, "memLimit", false)
	if result != nil {
		getContainerMetric(result, "namespace", "pod", "container", "memLimit")
	}

	query = `sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024`
	result = prometheus.MetricCollect(args, collectArgs, "memRequest", false)
	if result != nil {
		getContainerMetric(result, "namespace", "pod", "container", "memRequest")
	}

	query = `container_spec_cpu_shares{name!~"k8s_POD_.*"}`
	result = prometheus.MetricCollect(args, collectArgs, "conLabel", false)
	if result != nil {
		getContainerMetricString(result, "namespace", model.LabelName("pod"+args.LabelSuffix), model.LabelName("container"+args.LabelSuffix))
	}

	query = `kube_pod_container_info`
	result = prometheus.MetricCollect(args, collectArgs, "conInfo", false)
	if result != nil {
		getContainerMetricString(result, "namespace", "pod", "container")
	}

	//Pod metrics
	query = `kube_pod_info`
	result = prometheus.MetricCollect(args, collectArgs, "podInfo", false)
	if result != nil {
		getMidMetricString(result, "namespace", "pod", "Pod")
	}

	query = `kube_pod_labels`
	result = prometheus.MetricCollect(args, collectArgs, "podLabels", false)
	if result != nil {
		getMidMetricString(result, "namespace", "pod", "Pod")
	}

	query = `sum(kube_pod_container_status_restarts_total) by (pod,namespace,container)`
	result = prometheus.MetricCollect(args, collectArgs, "restarts", false)
	if result != nil {
		getContainerMetric(result, "namespace", "pod", "container", "restarts")
	}

	query = `sum(kube_pod_container_status_terminated) by (pod,namespace,container)`
	result = prometheus.MetricCollect(args, collectArgs, "powerState", false)
	if result != nil {
		getContainerMetric(result, "namespace", "pod", "container", "powerState")
	}

	query = `kube_pod_created`
	result = prometheus.MetricCollect(args, collectArgs, "podCreationTime", false)
	if result != nil {
		getMidMetric(result, "namespace", "pod", "creationTime", "Pod")
	}

	//Namespace metrics
	query = `kube_namespace_labels`
	result = prometheus.MetricCollect(args, collectArgs, "namespaceLabels", false)
	if result != nil {
		getNamespaceMetricString(result, "namespace")
	}

	query = `kube_namespace_annotations`
	result = prometheus.MetricCollect(args, collectArgs, "namespaceAnnotations", false)
	if result != nil {
		getNamespaceMetricString(result, "namespace")
	}

	query = `kube_limitrange`
	result = prometheus.MetricCollect(args, collectArgs, "nameSpaceLimitrange", false)
	if result != nil {
		getNamespacelimits(result, "namespace")
	}

	//Deployment metrics
	query = `kube_deployment_labels`
	result = prometheus.MetricCollect(args, collectArgs, "labels", false)
	if result != nil {
		getMidMetricString(result, "namespace", "deployment", "Deployment")
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_surge`
	result = prometheus.MetricCollect(args, collectArgs, "maxSurge", false)
	if result != nil {
		getMidMetric(result, "namespace", "deployment", "maxSurge", "Deployment")
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_unavailable`
	result = prometheus.MetricCollect(args, collectArgs, "maxUnavailable", false)
	if result != nil {
		getMidMetric(result, "namespace", "deployment", "maxUnavailable", "Deployment")
	}

	query = `kube_deployment_metadata_generation`
	result = prometheus.MetricCollect(args, collectArgs, "metadataGeneration", false)
	if result != nil {
		getMidMetric(result, "namespace", "deployment", "metadataGeneration", "Deployment")
	}

	query = `kube_deployment_created`
	result = prometheus.MetricCollect(args, collectArgs, "deploymentCreated", false)
	if result != nil {
		getMidMetric(result, "namespace", "deployment", "creationTime", "Deployment")
	}

	//ReplicaSet metrics
	query = `kube_replicaset_labels`
	result = prometheus.MetricCollect(args, collectArgs, "replicaSetLabels", false)
	if result != nil {
		getMidMetricString(result, "namespace", "replicaset", "ReplicaSet")
	}

	query = `kube_replicaset_created`
	result = prometheus.MetricCollect(args, collectArgs, "replicaSetCreated", false)
	if result != nil {
		getMidMetric(result, "namespace", "replicaset", "creationTime", "ReplicaSet")
	}

	//ReplicationController metrics
	query = `kube_replicationcontroller_created`
	result = prometheus.MetricCollect(args, collectArgs, "replicationControllerCreated", false)
	if result != nil {
		getMidMetric(result, "namespace", "replicationcontroller", "creationTime", "ReplicationController")
	}

	//DaemonSet metrics
	query = `kube_daemonset_labels`
	result = prometheus.MetricCollect(args, collectArgs, "daemonSetLabels", false)
	if result != nil {
		getMidMetricString(result, "namespace", "daemonset", "DaemonSet")
	}

	query = `kube_daemonset_created`
	result = prometheus.MetricCollect(args, collectArgs, "daemonSetCreated", false)
	if result != nil {
		getMidMetric(result, "namespace", "daemonset", "creationTime", "DaemonSet")
	}

	//StatefulSet metrics
	query = `kube_statefulset_labels`
	result = prometheus.MetricCollect(args, collectArgs, "statefulSetLabels", false)
	if result != nil {
		getMidMetricString(result, "namespace", "statefulset", "StatefulSet")
	}

	query = `kube_statefulset_created`
	result = prometheus.MetricCollect(args, collectArgs, "statefulSetCreated", false)
	if result != nil {
		getMidMetric(result, "namespace", "statefulset", "creationTime", "StatefulSet")
	}

	//Job metrics
	query = `kube_job_info * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result = prometheus.MetricCollect(args, collectArgs, "jobInfo", false)
	if result != nil {
		getMidMetricString(result, "namespace", "job_name", "Job")
	}

	query = `kube_job_labels * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result = prometheus.MetricCollect(args, collectArgs, "jobLabel", false)
	if result != nil {
		getMidMetricString(result, "namespace", "job_name", "Job")
	}

	query = `kube_job_spec_completions * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result = prometheus.MetricCollect(args, collectArgs, "jobSpecCompletions", false)
	if result != nil {
		getMidMetric(result, "namespace", "job_name", "specCompletions", "Job")
	}

	query = `kube_job_spec_parallelism * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result = prometheus.MetricCollect(args, collectArgs, "jobSpecParallelism", false)
	if result != nil {
		getMidMetric(result, "namespace", "job_name", "specParallelism", "Job")
	}

	query = `kube_job_status_completion_time * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result = prometheus.MetricCollect(args, collectArgs, "jobStatusCompletionTime", false)
	if result != nil {
		getMidMetric(result, "namespace", "job_name", "statusCompletionTime", "Job")
	}

	query = `kube_job_status_start_time * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result = prometheus.MetricCollect(args, collectArgs, "jobStatusStartTime", false)
	if result != nil {
		getMidMetric(result, "namespace", "job_name", "statusStartTime", "Job")
	}

	query = `kube_job_created`
	result = prometheus.MetricCollect(args, collectArgs, "jobCreated", false)
	if result != nil {
		getMidMetric(result, "namespace", "job", "creationTime", "Job")
	}

	//CronJob metrics
	query = `kube_cronjob_labels`
	result = prometheus.MetricCollect(args, collectArgs, "cronJobLabels", false)
	if result != nil {
		getMidMetricString(result, "namespace", "cronjob", "CronJob")
	}

	query = `kube_cronjob_info`
	result = prometheus.MetricCollect(args, collectArgs, "cronJobInfo", false)
	if result != nil {
		getMidMetricString(result, "namespace", "cronjob", "CronJob")
	}

	query = `kube_cronjob_next_schedule_time`
	result = prometheus.MetricCollect(args, collectArgs, "cronJobNextScheduleTime", false)
	if result != nil {
		getMidMetric(result, "namespace", "cronjob", "nextScheduleTime", "CronJob")
	}

	query = `kube_cronjob_status_last_schedule_time`
	result = prometheus.MetricCollect(args, collectArgs, "cronJobStatusLastScheduleTime", false)
	if result != nil {
		getMidMetric(result, "namespace", "cronjob", "lastScheduleTime", "CronJob")
	}

	query = `kube_cronjob_status_active`
	result = prometheus.MetricCollect(args, collectArgs, "cronJobStatusActive", false)
	if result != nil {
		getMidMetric(result, "namespace", "cronjob", "statusActive", "CronJob")
	}

	query = `kube_cronjob_created`
	result = prometheus.MetricCollect(args, collectArgs, "cronJobCreated", false)
	if result != nil {
		getMidMetric(result, "namespace", "cronjob", "creationTime", "CronJob")
	}

	//HPA metrics
	query = `kube_hpa_labels`
	result = prometheus.MetricCollect(args, collectArgs, "hpaLabels", false)
	if result != nil {
		getHPAMetricString(result, "namespace", "hpa", args)
	}

	//Current size workloads
	currentSizeWrite, err := os.Create("./data/container/currentSize.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("entity=" + entityKind + " message=" + err.Error())
		return
	}
	fmt.Fprintf(currentSizeWrite, "cluster,namespace,entity_name,entity_type,container,Datetime,Auto Scaling - In Service Instances\n")

	query = `kube_replicaset_spec_replicas`
	result = prometheus.MetricCollect(args, collectArgs, "replicaSetSpecReplicas", false)
	if result != nil {
		getMidMetric(result, "namespace", "replicaset", "currentSize", "ReplicaSet")
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "replicaset", args, "ReplicaSet")

	query = `kube_replicationcontroller_spec_replicas`
	result = prometheus.MetricCollect(args, collectArgs, "replicationcontroller_spec_replicas", false)
	if result != nil {
		getMidMetric(result, "namespace", "replicationcontroller", "currentSize", "ReplicationController")
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "replicationcontroller", args, "ReplicationController")

	query = `kube_daemonset_status_number_available`
	result = prometheus.MetricCollect(args, collectArgs, "daemonSetStatusNumberAvailable", false)
	if result != nil {
		getMidMetric(result, "namespace", "daemonset", "currentSize", "DaemonSet")
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "daemonset", args, "DaemonSet")

	query = `kube_statefulset_replicas`
	result = prometheus.MetricCollect(args, collectArgs, "statefulSetReplicas", false)
	if result != nil {
		getMidMetric(result, "namespace", "statefulset", "currentSize", "StatefulSet")
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "statefulset", args, "StatefulSet")

	query = `kube_job_spec_parallelism`
	result = prometheus.MetricCollect(args, collectArgs, "jobSpecParallelism", false)
	if result != nil {
		getMidMetric(result, "namespace", "job_name", "currentSize", "Job")
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "job_name", args, "Job")

	query = `sum(max(kube_job_spec_parallelism) by (namespace,job_name) * on (namespace,job_name) group_right max(kube_job_owner) by (namespace, job_name, owner_name)) by (owner_name, namespace)`
	result = prometheus.MetricCollect(args, collectArgs, "cronJobSpecParallelism", false)
	if result != nil {
		getMidMetric(result, "namespace", "owner_name", "currentSize", "CronJob")
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "owner_name", args, "CronJob")

	query = `sum(max(kube_replicaset_spec_replicas) by (namespace,replicaset) * on (namespace,replicaset) group_right max(kube_replicaset_owner) by (namespace, replicaset, owner_name)) by (owner_name, namespace)`
	result = prometheus.MetricCollect(args, collectArgs, "replicaSetSpecReplicas", false)
	if result != nil {
		getMidMetric(result, "namespace", "owner_name", "currentSize", "Deployment")
	}
	writeWorkloadMid(currentSizeWrite, result, "namespace", "owner_name", args, "Deployment")

	currentSizeWrite.Close()

	writeAttributes(args)
	writeConfig(args)

	queryPrefix := ``
	querySuffix := ``
	if args.LabelSuffix != "" {
		queryPrefix = `label_replace(`
		querySuffix = `, "pod", "$1", "pod_name", "(.*)")`
	}

	//Container workloads
	query = queryPrefix + `round(sum(irate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[5m])) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)*1000,1)` + querySuffix
	getWorkload("cpu_mCores_workload", "CPU Utilization in mCores", query, "max", args)
	getWorkload("cpu_mCores_workload", "Prometheus CPU Utilization in mCores", query, "avg", args)

	query = queryPrefix + `sum(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)` + querySuffix
	getWorkload("mem_workload", "Raw Mem Utilization", query, "max", args)
	getWorkload("mem_workload", "Prometheus Raw Mem Utilization", query, "avg", args)

	query = queryPrefix + `sum(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)` + querySuffix
	getWorkload("rss_workload", "Actual Memory Utilization", query, "max", args)
	getWorkload("rss_workload", "Prometheus Actual Memory Utilization", query, "avg", args)

	query = queryPrefix + `sum(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)` + querySuffix
	getWorkload("disk_workload", "Raw Disk Utilization", query, "max", args)
	getWorkload("disk_workload", "Prometheus Raw Disk Utilization", query, "avg", args)

	if args.LabelSuffix != "" {
		queryPrefix = `label_replace(`
		querySuffix = `, "container_name", "$1", "container", "(.*)")`
	}
	query = queryPrefix + `sum(irate(kube_pod_container_status_restarts_total{name!~"k8s_POD_.*"}[1h])) by (instance,pod,namespace,container)` + querySuffix
	getWorkload("restarts", "Restarts", query, "max", args)

	if args.LabelSuffix == "" {
		query = `kube_hpa_status_condition{status="true",condition="ScalingLimited"}`
	} else {
		query = `kube_hpa_status_condition{status="ScalingLimited",condition="true"}`
	}
	getHPAWorkload("condition_scaling_limited", "Scaling Limited", query, args)

	//HPA workloads
	query = `kube_hpa_spec_max_replicas`
	getHPAWorkload("max_replicas", "Auto Scaling - Maximum Size", query, args)

	query = `kube_hpa_spec_min_replicas`
	getHPAWorkload("min_replicas", "Auto Scaling - Minimum Size", query, args)

	query = `kube_hpa_status_current_replicas`
	getHPAWorkload("current_replicas", "Auto Scaling - Total Instances", query, args)

}
