//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"fmt"
	"os"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
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
	var err error

	var podOwners = map[string]string{}
	var podOwnersKind = map[string]string{}
	var replicaSetOwners = map[string]string{}
	var jobOwners = map[string]string{}

	range5Min := common.TimeRange(args, historyInterval)

	//querys gathering hierarchy information for the containers
	query = `sum(kube_pod_owner{owner_name!="<none>"}) by (namespace, pod, owner_name, owner_kind)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.ErrorLogger.Println("metric=pods query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=pods query=" + query + " message=" + err.Error())
		return
	}

	rslt = result.(model.Matrix)
	for i := 0; i < rslt.Len(); i++ {
		podOwners[string(rslt[i].Metric["pod"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_name"])
		podOwnersKind[string(rslt[i].Metric["pod"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_kind"])
	}

	query = `sum(kube_replicaset_owner{owner_name!="<none>"}) by (namespace, replicaset, owner_name)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=replicasets query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicasets query=" + query + " message=" + err.Error())
	} else {
		rslt = result.(model.Matrix)
		for i := 0; i < rslt.Len(); i++ {
			replicaSetOwners[string(rslt[i].Metric["replicaset"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_name"])
			args.Deployments = true
		}
	}

	query = `sum(kube_job_owner{owner_name!="<none>"}) by (namespace, job_name, owner_name)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=jobs query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobs query=" + query + " message=" + err.Error())
	} else {
		rslt = result.(model.Matrix)
		for i := 0; i < rslt.Len(); i++ {
			jobOwners[string(rslt[i].Metric["job_name"])+"__"+string(rslt[i].Metric["namespace"])] = string(rslt[i].Metric["owner_name"])
			args.CronJobs = true
		}
	}

	query = `max(kube_pod_container_info) by (container, pod, namespace)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.ErrorLogger.Println("metric=containers query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=containers query=" + query + " message=" + err.Error())
		return
	}

	rslt = result.(model.Matrix)

	var currentOwner string

	if !args.CronJobs {
		args.InfoLogger.Println("No CronJobs found")
		fmt.Println("[Info] No CronJobs found")
	}
	if !args.Deployments {
		args.InfoLogger.Println("No Deployments found")
		fmt.Println("[Info] No Deployments found")
	}

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
		fmt.Println("[DEBUG] message=Dump of Systesms structure\n" + tempString)
	}

	//Container metrics
	query = `container_spec_memory_limit_bytes{name!~"k8s_POD_.*"}/1024/1024`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=memory query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=memory query=" + query + " message=" + err.Error())
	} else {
		if args.LabelSuffix == "" && getContainerMetric(result, "namespace", "pod", "container", "memory") {
			//Don't do anything
		} else if getContainerMetric(result, "namespace", "pod_name", "container_name", "memory") {
			args.LabelSuffix = "_name"
		}
	}

	query = `sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=cpuLimit query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cpuLimit query=" + query + " message=" + err.Error())
	} else {
		getContainerMetric(result, "namespace", "pod", "container", "cpuLimit")
	}

	query = `sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=cpuRequest query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cpuRequest query=" + query + " message=" + err.Error())
	} else {
		getContainerMetric(result, "namespace", "pod", "container", "cpuRequest")
	}

	query = `sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=memLimit query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=openshift_clustememLimitrresourcequota_selector query=" + query + " message=" + err.Error())
	} else {
		getContainerMetric(result, "namespace", "pod", "container", "memLimit")
	}

	query = `sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=memRequest query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=memRequest query=" + query + " message=" + err.Error())
	} else {
		getContainerMetric(result, "namespace", "pod", "container", "memRequest")
	}

	query = `container_spec_cpu_shares{name!~"k8s_POD_.*"}`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=conLabel query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=conLabel query=" + query + " message=" + err.Error())
	} else {
		getContainerMetricString(result, "namespace", model.LabelName("pod"+args.LabelSuffix), model.LabelName("container"+args.LabelSuffix))
	}

	query = `kube_pod_container_info`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=conInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=conInfo query=" + query + " message=" + err.Error())
	} else {
		getContainerMetricString(result, "namespace", "pod", "container")
	}

	//Pod metrics
	query = `kube_pod_info`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=podInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podInfo query=" + query + " message=" + err.Error())
	} else {
		getMidMetricString(result, "namespace", "pod", "Pod")
	}

	query = `kube_pod_labels`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=podLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetricString(result, "namespace", "pod", "Pod")
	}

	query = `sum(kube_pod_container_status_restarts_total) by (pod,namespace,container)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=restarts query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=restarts query=" + query + " message=" + err.Error())
	} else {
		getContainerMetric(result, "namespace", "pod", "container", "restarts")
	}

	query = `sum(kube_pod_container_status_terminated) by (pod,namespace,container)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=powerState query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=powerState query=" + query + " message=" + err.Error())
	} else {
		getContainerMetric(result, "namespace", "pod", "container", "powerState")
	}

	query = `kube_pod_created`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=podCreationTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podCreationTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "pod", "creationTime", "Pod")
	}

	//Namespace metrics
	query = `kube_namespace_labels`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=namespaceLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=namespaceLabels query=" + query + " message=" + err.Error())
	} else {
		getNamespaceMetricString(result, "namespace")
	}

	query = `kube_namespace_annotations`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=namespaceAnnotations query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=namespaceAnnotations query=" + query + " message=" + err.Error())
	} else {
		getNamespaceMetricString(result, "namespace")
	}

	query = `kube_limitrange`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=nameSpaceLimitrange query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=nameSpaceLimitrange query=" + query + " message=" + err.Error())
	} else {
		getNamespacelimits(result, "namespace")
	}

	//Deployment metrics
	query = `kube_deployment_labels`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=labels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=labels query=" + query + " message=" + err.Error())
	} else {
		getMidMetricString(result, "namespace", "deployment", "Deployment")
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_surge`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=maxSurge query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=maxSurge query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "deployment", "maxSurge", "Deployment")
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_unavailable`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=maxUnavailable query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=maxUnavailable query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "deployment", "maxUnavailable", "Deployment")
	}

	query = `kube_deployment_metadata_generation`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=metadataGeneration query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=metadataGeneration query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "deployment", "metadataGeneration", "Deployment")
	}

	query = `kube_deployment_created`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=deploymentCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=deploymentCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "deployment", "creationTime", "Deployment")
	}

	//ReplicaSet metrics
	query = `kube_replicaset_labels`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=replicaSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicaSetLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetricString(result, "namespace", "replicaset", "ReplicaSet")
	}

	query = `kube_replicaset_created`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=replicaSetCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicaSetCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "replicaset", "creationTime", "ReplicaSet")
	}

	//ReplicationController metrics
	query = `kube_replicationcontroller_created`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=replicationControllerCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicationControllerCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "replicationcontroller", "creationTime", "ReplicationController")
	}

	//DaemonSet metrics
	query = `kube_daemonset_labels`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=daemonSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=daemonSetLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetricString(result, "namespace", "daemonset", "DaemonSet")
	}

	query = `kube_daemonset_created`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=daemonSetCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=daemonSetCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "daemonset", "creationTime", "DaemonSet")
	}

	//StatefulSet metrics
	query = `kube_statefulset_labels`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=statefulSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=statefulSetLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetricString(result, "namespace", "statefulset", "StatefulSet")
	}

	query = `kube_statefulset_created`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=statefulSetCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=statefulSetCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "statefulset", "creationTime", "StatefulSet")
	}

	//Job metrics
	query = `kube_job_info * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=jobInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobInfo query=" + query + " message=" + err.Error())
	} else {
		getMidMetricString(result, "namespace", "job_name", "Job")
	}

	query = `kube_job_labels * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=jobLabel query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobLabel query=" + query + " message=" + err.Error())
	} else {
		getMidMetricString(result, "namespace", "job_name", "Job")
	}

	query = `kube_job_spec_completions * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=jobSpecCompletions query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobSpecCompletions query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "job_name", "specCompletions", "Job")
	}

	query = `kube_job_spec_parallelism * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=jobSpecParallelism query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobSpecParallelism query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "job_name", "specParallelism", "Job")
	}

	query = `kube_job_status_completion_time * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=jobStatusCompletionTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobStatusCompletionTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "job_name", "statusCompletionTime", "Job")
	}

	query = `kube_job_status_start_time * on (namespace,job_name) group_left (owner_name) max(kube_job_owner) by (namespace, job_name, owner_name)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=jobStatusStartTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobStatusStartTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "job_name", "statusStartTime", "Job")
	}

	query = `kube_job_created`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=jobCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "job", "creationTime", "Job")
	}

	//CronJob metrics
	query = `kube_cronjob_labels`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetricString(result, "namespace", "cronjob", "CronJob")
	}

	query = `kube_cronjob_info`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobInfo query=" + query + " message=" + err.Error())
	} else {
		getMidMetricString(result, "namespace", "cronjob", "CronJob")
	}

	query = `kube_cronjob_next_schedule_time`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobNextScheduleTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobNextScheduleTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "cronjob", "nextScheduleTime", "CronJob")
	}

	query = `kube_cronjob_status_last_schedule_time`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobStatusLastScheduleTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobStatusLastScheduleTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "cronjob", "lastScheduleTime", "CronJob")
	}

	query = `kube_cronjob_status_active`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobStatusActive query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobStatusActive query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "cronjob", "statusActive", "CronJob")
	}

	query = `kube_cronjob_created`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "namespace", "cronjob", "creationTime", "CronJob")
	}

	//HPA metrics
	query = `kube_hpa_labels`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=hpaLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=hpaLabels query=" + query + " message=" + err.Error())
	} else {
		getHPAMetricString(result, "namespace", "hpa", args)
	}

	//Current size workloads
	currentSizeWrite, err := os.Create("./data/container/currentSize.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
	} else {
		fmt.Fprintf(currentSizeWrite, "cluster,namespace,entity_name,entity_type,container,Datetime,Auto Scaling - In Service Instances\n")

		query = `kube_replicaset_spec_replicas`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=replicaSetSpecReplicas query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=replicaSetSpecReplicas query=" + query + " message=" + err.Error())
		} else {
			getMidMetric(result, "namespace", "replicaset", "currentSize", "ReplicaSet")
			writeWorkloadMid(currentSizeWrite, result, "namespace", "replicaset", args, "ReplicaSet")
		}

		query = `kube_replicationcontroller_spec_replicas`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=replicationcontroller_spec_replicas query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=replicationcontroller_spec_replicas query=" + query + " message=" + err.Error())
		} else {
			getMidMetric(result, "namespace", "replicationcontroller", "currentSize", "ReplicationController")
			writeWorkloadMid(currentSizeWrite, result, "namespace", "replicationcontroller", args, "ReplicationController")
		}

		query = `kube_daemonset_status_number_available`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=daemonSetStatusNumberAvailable query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=daemonSetStatusNumberAvailable query=" + query + " message=" + err.Error())
		} else {
			getMidMetric(result, "namespace", "daemonset", "currentSize", "DaemonSet")
			writeWorkloadMid(currentSizeWrite, result, "namespace", "daemonset", args, "DaemonSet")
		}

		query = `kube_statefulset_replicas`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=statefulSetReplicas query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=statefulSetReplicas query=" + query + " message=" + err.Error())
		} else {
			getMidMetric(result, "namespace", "statefulset", "currentSize", "StatefulSet")
			writeWorkloadMid(currentSizeWrite, result, "namespace", "statefulset", args, "StatefulSet")
		}

		query = `kube_job_spec_parallelism`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=jobSpecParallelism query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=jobSpecParallelism query=" + query + " message=" + err.Error())
		} else {
			getMidMetric(result, "namespace", "job_name", "currentSize", "Job")
			writeWorkloadMid(currentSizeWrite, result, "namespace", "job_name", args, "Job")
		}

		query = `sum(max(kube_job_spec_parallelism) by (namespace,job_name) * on (namespace,job_name) group_right max(kube_job_owner) by (namespace, job_name, owner_name)) by (owner_name, namespace)`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=cronJobSpecParallelism query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cronJobSpecParallelism query=" + query + " message=" + err.Error())
		} else {
			getMidMetric(result, "namespace", "owner_name", "currentSize", "CronJob")
			writeWorkloadMid(currentSizeWrite, result, "namespace", "owner_name", args, "CronJob")
		}

		query = `sum(max(kube_replicaset_spec_replicas) by (namespace,replicaset) * on (namespace,replicaset) group_right max(kube_replicaset_owner) by (namespace, replicaset, owner_name)) by (owner_name, namespace)`
		result, err = common.MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=replicaSetSpecReplicas query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=replicaSetSpecReplicas query=" + query + " message=" + err.Error())
		} else {
			getMidMetric(result, "namespace", "owner_name", "currentSize", "Deployment")
			writeWorkloadMid(currentSizeWrite, result, "namespace", "owner_name", args, "Deployment")
		}

		currentSizeWrite.Close()
	}

	writeAttributes(args)
	writeConfig(args)

	queryPrefix := ``
	querySuffix := ``
	if args.LabelSuffix != "" {
		queryPrefix = `label_replace(`
		querySuffix = `, "pod", "$1", "pod_name", "(.*)")`
	}

	//Container workloads
	query = queryPrefix + `round(max(irate(container_cpu_usage_seconds_total{name!~"k8s_POD_.*"}[` + args.SampleRateString + `m])) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)*1000,1)` + querySuffix
	getWorkload("cpu_mCores_workload", "CPU Utilization in mCores", query, "max", args)
	getWorkload("cpu_mCores_workload", "Prometheus CPU Utilization in mCores", query, "avg", args)

	query = queryPrefix + `max(container_memory_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)` + querySuffix
	getWorkload("mem_workload", "Raw Mem Utilization", query, "max", args)
	getWorkload("mem_workload", "Prometheus Raw Mem Utilization", query, "avg", args)

	query = queryPrefix + `max(container_memory_rss{name!~"k8s_POD_.*"}) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)` + querySuffix
	getWorkload("rss_workload", "Actual Memory Utilization", query, "max", args)
	getWorkload("rss_workload", "Prometheus Actual Memory Utilization", query, "avg", args)

	query = queryPrefix + `max(container_fs_usage_bytes{name!~"k8s_POD_.*"}) by (instance,pod` + args.LabelSuffix + `,namespace,container` + args.LabelSuffix + `)` + querySuffix
	getWorkload("disk_workload", "Raw Disk Utilization", query, "max", args)
	getWorkload("disk_workload", "Prometheus Raw Disk Utilization", query, "avg", args)

	if args.LabelSuffix != "" {
		queryPrefix = `label_replace(`
		querySuffix = `, "container_name", "$1", "container", "(.*)")`
	}
	query = queryPrefix + `max(irate(kube_pod_container_status_restarts_total{name!~"k8s_POD_.*"}[` + args.SampleRateString + `m])) by (instance,pod,namespace,container)` + querySuffix
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
