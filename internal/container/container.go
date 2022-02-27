// Package container collects data related to containers and formats into json files to send to Densify.
package container

import (
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"runtime"
	"strings"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

var systems = map[string]*datamodel.Namespace{}

const (
	ownerKindKey        = "owner_kind"
	ownerNameKey        = "owner_name"
	podEntityKey        = "Pods"
	podMetricKey        = "pod"
	replicaSetEntityKey = "ReplicaSet"
	replicaSetMetricKey = "replicaset"
	jobEntityKey        = "Job"
	jobMetricKey        = "job_name"
	containerIdKey      = "container_id"
	imageIdKey          = "image_id"
)

var ownerKindFilter = []string{ownerKindKey}
var ownerNameFilter = []string{ownerNameKey}

//getContainerMetric is used to parse the results from Prometheus related to Container Entities and store them in the systems data structure.
func getContainerMetric(result model.Value, pod, container model.LabelName, metric string) bool {
	mat, ok := result.(model.Matrix)
	if !ok {
		return false
	}
	// Loop through the different entities in the results
	n := mat.Len()
	for i := 0; i < n; i++ {
		//Validate that the data contains the namespace label with value and check it exists in our systems structure.
		nsValue, ok := mat[i].Metric[common.NamespaceKey]
		if !ok {
			continue
		}
		namespaceValue := string(nsValue)
		if _, ok = systems[namespaceValue]; !ok {
			continue
		}
		//Validate that the data contains the pod label with value and check it exists in our systems structure
		pValue, ok := mat[i].Metric[pod]
		if !ok {
			continue
		}
		podValue := string(pValue)
		if _, ok = systems[namespaceValue].Entities[podEntityKey][podValue]; !ok {
			continue
		}
		//Validate that the data contains the container label with value and check it exists in our systems structure
		containerNameValue, ok := mat[i].Metric[container]
		if !ok {
			continue
		}
		var container *datamodel.Container
		if container, ok = systems[namespaceValue].Entities[podEntityKey][podValue].Containers[string(containerNameValue)]; !ok {
			continue
		}

		//Check which metric this is for and update the corresponding variable for this container in the system data structure
		switch metric {
		case "limits":
			switch mat[i].Metric["resource"] {
			case "memory":
				_ = container.MemLimit.AppendSampleStreamWithValue(mat[i], "", memConvert)
			case "cpu":
				_ = container.CpuLimit.AppendSampleStreamWithValue(mat[i], "", cpuConvert)
			}
		case "requests":
			switch mat[i].Metric["resource"] {
			case "memory":
				_ = container.MemRequest.AppendSampleStreamWithValue(mat[i], "", memConvert)
			case "cpu":
				_ = container.CpuRequest.AppendSampleStreamWithValue(mat[i], "", cpuConvert)
			}
		case "cpuLimit":
			_ = container.CpuLimit.AppendSampleStreamWithValue(mat[i], "", cpuConvert)
		case "cpuRequest":
			_ = container.CpuRequest.AppendSampleStreamWithValue(mat[i], "", cpuConvert)
		case "memLimit":
			_ = container.MemLimit.AppendSampleStreamWithValue(mat[i], "", memConvert)
		case "memRequest":
			_ = container.MemRequest.AppendSampleStreamWithValue(mat[i], "", memConvert)
		case "powerState":
			_ = container.PowerState.AppendSampleStreamWithValue(mat[i], "", boolConv)
		case "kube_pod_container_info":
			// filter out cases when container_id or image_id are not populated yet (may happen
			// in the phase of ContainerCreating), as these cause a lot of noise of diffs
			if _, ok = mat[i].Metric[containerIdKey]; !ok {
				continue
			}
			if _, ok = mat[i].Metric[imageIdKey]; !ok {
				continue
			}
			// if both are there, fallthrough
			fallthrough
		default:
			labels := datamodel.EnsureLabels(container.LabelMap, metric)
			_ = labels.AppendSampleStream(mat[i])
		}
	}
	return true
}

func cpuConv(value float64) float64 {
	return value * 1000
}

func memConv(value float64) float64 {
	return value / 1024 / 1024
}

var cpuConvert = &datamodel.Converter{VCF: cpuConv}
var memConvert = &datamodel.Converter{VCF: memConv}
var timeConv = datamodel.TimeStampConverter()
var boolConv = datamodel.BoolConverter()

//getmidMetric is used to parse the results from Prometheus related to mid Entities and store them in the systems data structure.
func getMidMetric(result model.Value, mid model.LabelName, metric, kind, query string) {
	mat, ok := result.(model.Matrix)
	if !ok {
		return
	}
	// Loop through the different entities in the results.
	n := mat.Len()
	for i := 0; i < n; i++ {
		//Validate that the data contains the namespace label with value and check it exists in our systems structure.
		nsVal, ok := mat[i].Metric[common.NamespaceKey]
		if !ok {
			continue
		}
		namespaceValue := string(nsVal)
		if _, ok = systems[namespaceValue]; !ok {
			continue
		}
		//Validate that the data contains the mid label with value and check it exists in our systems structure
		midVal, ok := mat[i].Metric[mid]
		if !ok {
			continue
		}
		midValue := string(midVal)
		if _, ok := systems[namespaceValue].Entities[kind][midValue]; !ok {
			continue
		}

		//Check which metric this is for and update the corresponding variable for this mid in the system data structure
		switch metric {
		case "label":
			labels := datamodel.EnsureLabels(systems[namespaceValue].Entities[kind][midValue].LabelMap, query)
			_ = labels.AppendSampleStream(mat[i])
		case "creationTime":
			_ = systems[namespaceValue].Entities[kind][midValue].CreationTime.AppendSampleStreamWithValue(mat[i], "", timeConv)
		case "kube_cronjob_next_schedule_time":
			_ = systems[namespaceValue].Entities[kind][midValue].NextSchedTime.AppendSampleStreamWithValue(mat[i], "", timeConv)
		case "kube_cronjob_status_active":
			_ = systems[namespaceValue].Entities[kind][midValue].StatusActive.AppendSampleStreamWithValue(mat[i], "", boolConv)
		case "kube_cronjob_status_last_schedule_time":
			_ = systems[namespaceValue].Entities[kind][midValue].LastSchedTime.AppendSampleStreamWithValue(mat[i], "", timeConv)
		case "kube_deployment_metadata_generation":
			_ = systems[namespaceValue].Entities[kind][midValue].MetadataGeneration.AppendSampleStreamWithValue(mat[i], "", nil)
		case "kube_deployment_spec_strategy_rollingupdate_max_surge":
			_ = systems[namespaceValue].Entities[kind][midValue].MaxSurge.AppendSampleStreamWithValue(mat[i], "", nil)
		case "kube_deployment_spec_strategy_rollingupdate_max_unavailable":
			_ = systems[namespaceValue].Entities[kind][midValue].MaxUnavailable.AppendSampleStreamWithValue(mat[i], "", nil)
		case "kube_job_status_completion_time":
			_ = systems[namespaceValue].Entities[kind][midValue].CompletionTime.AppendSampleStreamWithValue(mat[i], "", timeConv)
		case "kube_job_spec_completions":
			_ = systems[namespaceValue].Entities[kind][midValue].Completions.AppendSampleStreamWithValue(mat[i], "", nil)
		case "kube_job_spec_parallelism":
			_ = systems[namespaceValue].Entities[kind][midValue].Parallelism.AppendSampleStreamWithValue(mat[i], "", nil)
		}
	}
}

//getNamespaceMetric is used to parse the label based results from Prometheus related to Namespace Entities and store them in the systems data structure.
func getNamespaceMetric(result model.Value, query string) {
	mat, ok := result.(model.Matrix)
	if !ok {
		return
	}
	// Loop through the different entities in the results.
	n := mat.Len()
	for i := 0; i < n; i++ {
		//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
		nsValue, ok := mat[i].Metric[common.NamespaceKey]
		if !ok {
			continue
		}
		namespaceValue := string(nsValue)
		if _, ok := systems[namespaceValue]; !ok {
			continue
		}
		labels := &datamodel.Labels{}
		systems[namespaceValue].LabelMap[query] = labels
		_ = labels.AppendSampleStream(mat[i])
	}
}

//Metrics function to collect data related to containers.
func Metrics(args *common.Parameters) {
	//Setup variables used in the code.
	var query, labelSuffix string
	var result model.Value
	var err error
	var mem runtime.MemStats

	if args.Debug {
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}

	//query to get info about what containers exist. Using this query that is part of kube-state-metrics means very likely will miss collecting anything related to the k8s_pause container side cars as don't tend to show up in KSM.
	query = `kube_pod_container_info`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.ErrorLogger.Println("metric=containers query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=containers query=" + query + " message=" + err.Error())
		return
	} else {
		//Add containers to pods structure
		mat := result.(model.Matrix)
		n := mat.Len()
		for i := 0; i < n; i++ {
			// Get the container, pod and namespace names.
			containerName := string(mat[i].Metric["container"])
			podName := string(mat[i].Metric[podMetricKey])
			namespaceName := string(mat[i].Metric[common.NamespaceKey])
			//check if already have setup namespace, pod in system structure and if not add them.
			if _, ok := systems[namespaceName]; !ok {
				systems[namespaceName] = &datamodel.Namespace{LabelMap: make(datamodel.LabelMap), Entities: make(map[string]map[string]*datamodel.MidLevel)}
				systems[namespaceName].Entities[podEntityKey] = make(map[string]*datamodel.MidLevel)
			}
			if _, ok := systems[namespaceName].Entities[podEntityKey][podName]; !ok {
				systems[namespaceName].Entities[podEntityKey][podName] = newMidLevel()
			}
			//If the container doesn't exist then look to add and call getContainerMetric to set labels.
			if _, ok := systems[namespaceName].Entities[podEntityKey][podName].Containers[containerName]; !ok {
				systems[namespaceName].Entities[podEntityKey][podName].Containers[containerName] = newContainer()
				getContainerMetric(result, "pod", "container", query)
			}
		}
	}

	// Create any entities based on pods that are owned by them.
	// This will create replicasets, jobs, daemonsets, etc.
	if err = getMidLevelOwner(args, "kube_pod_owner", podEntityKey, podMetricKey); err != nil {
		return
	}

	// Check for any replicaset that are owned by deployments.
	_ = getMidLevelOwner(args, "kube_replicaset_owner", replicaSetEntityKey, replicaSetMetricKey)

	//Check for any jobs that are owned by Cronjobs
	_ = getMidLevelOwner(args, "kube_job_owner", jobEntityKey, jobMetricKey)

	if args.Debug {
		args.DebugLogger.Println("message=Collecting Container Metrics")
		fmt.Println("[DEBUG] message=Collecting Container Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}

	//query for the Container resource limits (cpu and memory) if we get no results for the newer combined metric then will fall back to older metrics which get just CPU and Memory.
	query = `kube_pod_container_resource_limits`
	result, err = common.MetricCollect(args, query)
	mat := result.(model.Matrix)
	n := mat.Len()
	if n == 0 {
		query = `kube_pod_container_resource_limits_cpu_cores`
		result, err = common.MetricCollect(args, query)
		if err != nil {
			args.WarnLogger.Println("metric=cpuLimit query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cpuLimit query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "cpuLimit")
		}

		query = `kube_pod_container_resource_limits_memory_bytes`
		result, err = common.MetricCollect(args, query)
		if err != nil {
			args.WarnLogger.Println("metric=memLimit query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=memLimit query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "memLimit")
		}
	} else {
		getContainerMetric(result, "pod", "container", "limits")
	}

	//Query for the Container resource requests (cpu and memory) if we get no results for the newer combined metric then will fall back to older metrics which get just CPU and Memory.
	query = `kube_pod_container_resource_requests`
	result, err = common.MetricCollect(args, query)
	mat = result.(model.Matrix)
	n = mat.Len()
	if n == 0 {
		query = `kube_pod_container_resource_requests_cpu_cores`
		result, err = common.MetricCollect(args, query)
		if err != nil {
			args.WarnLogger.Println("metric=cpuRequest query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cpuRequest query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "cpuRequest")
		}

		query = `kube_pod_container_resource_requests_memory_bytes`
		result, err = common.MetricCollect(args, query)
		if err != nil {
			args.WarnLogger.Println("metric=memRequest query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=memRequest query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "memRequest")
		}
	} else {
		getContainerMetric(result, "pod", "container", "requests")
	}

	// Getting info from cAdvisor metric that will be stored as labels.
	query = `container_spec_cpu_shares`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=conLabel query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=conLabel query=" + query + " message=" + err.Error())
	} else {
		//Based on setup older versions of Prometheus may have the _name appended to metric fields so we are checking if we see results using new or old method and set labelSuffix if need for use in other queries.
		if getContainerMetric(result, "pod", "container", query) {
			//Don't do anything
		} else if getContainerMetric(result, "pod_name", "container_name", query) {
			labelSuffix = "_name"
		}
	}

	//Pod metrics
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Pod Metrics")
		fmt.Println("[DEBUG] message=Collecting Pod Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}
	query = `kube_pod_info`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=podInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podInfo query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "pod", "label", podEntityKey, query)
	}

	query = `kube_pod_labels`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=podLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "pod", "label", podEntityKey, query)
	}

	//Depending on version of KSM will determine what metric need to use to see if containers are terminated or not for there state.
	query = `kube_pod_container_status_terminated`
	result, err = common.MetricCollect(args, query)
	mat = result.(model.Matrix)
	n = mat.Len()
	if n == 0 {
		query = `kube_pod_container_status_terminated_reason`
		result, err = common.MetricCollect(args, query)
		if err != nil {
			args.WarnLogger.Println("metric=powerState query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=powerState query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "powerState")
		}
	} else {
		if err != nil {
			args.WarnLogger.Println("metric=powerState query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=powerState query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "powerState")
		}
	}

	query = `kube_pod_created`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=podCreationTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podCreationTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "pod", "creationTime", "Pod", query)
	}

	//Namespace metrics
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Namespace Metrics")
		fmt.Println("[DEBUG] message=Collecting Namespace Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}
	query = `kube_namespace_labels`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=namespaceLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=namespaceLabels query=" + query + " message=" + err.Error())
	} else {
		getNamespaceMetric(result, query)
	}

	query = `kube_namespace_annotations`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=namespaceAnnotations query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=namespaceAnnotations query=" + query + " message=" + err.Error())
	} else {
		getNamespaceMetric(result, query)
	}

	//Deployment metrics
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Deployment Metrics")
		fmt.Println("[DEBUG] message=Collecting Deployment Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}
	query = `kube_deployment_labels`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=labels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=labels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "deployment", "label", "Deployment", query)
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_surge`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=maxSurge query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=maxSurge query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "deployment", query, "Deployment", query)
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_unavailable`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=maxUnavailable query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=maxUnavailable query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "deployment", query, "Deployment", query)
	}

	query = `kube_deployment_metadata_generation`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=metadataGeneration query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=metadataGeneration query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "deployment", query, "Deployment", query)
	}

	query = `kube_deployment_created`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=deploymentCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=deploymentCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "deployment", "creationTime", "Deployment", query)
	}

	//ReplicaSet metrics
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Replica Set Metrics")
		fmt.Println("[DEBUG] message=Collecting Replica Set Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}
	query = `kube_replicaset_labels`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=replicaSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicaSetLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "replicaset", "label", "ReplicaSet", query)
	}

	query = `kube_replicaset_created`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=replicaSetCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicaSetCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "replicaset", "creationTime", "ReplicaSet", query)
	}

	//ReplicationController metrics
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Replication Controller Metrics")
		fmt.Println("[DEBUG] message=Collecting Replication Controller Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}
	query = `kube_replicationcontroller_created`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=replicationControllerCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicationControllerCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "replicationcontroller", "creationTime", "ReplicationController", query)
	}

	//DaemonSet metrics
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Daemon Set Metrics")
		fmt.Println("[DEBUG] message=Collecting Daemon Set Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}
	query = `kube_daemonset_labels`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=daemonSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=daemonSetLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "daemonset", "label", "DaemonSet", query)
	}

	query = `kube_daemonset_created`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=daemonSetCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=daemonSetCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "daemonset", "creationTime", "DaemonSet", query)
	}

	//StatefulSet metrics
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Stateful Set Metrics")
		fmt.Println("[DEBUG] message=Collecting Stateful Set Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}
	query = `kube_statefulset_labels`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=statefulSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=statefulSetLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "statefulset", "label", "StatefulSet", query)
	}

	query = `kube_statefulset_created`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=statefulSetCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=statefulSetCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "statefulset", "creationTime", "StatefulSet", query)
	}

	//Job metrics
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Job Metrics")
		fmt.Println("[DEBUG] message=Collecting Job Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}
	query = `kube_job_info`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobInfo query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", "label", "Job", query)
	}

	query = `kube_job_labels`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobLabel query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobLabel query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", "label", "Job", query)
	}

	query = `kube_job_spec_completions`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobSpecCompletions query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobSpecCompletions query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", query, "Job", query)
	}

	query = `kube_job_spec_parallelism`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobSpecParallelism query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobSpecParallelism query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", query, "Job", query)
	}

	query = `kube_job_status_completion_time`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobStatusCompletionTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobStatusCompletionTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", query, "Job", query)
	}

	query = `kube_job_status_start_time`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobStatusStartTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobStatusStartTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", "creationTime", "Job", query)
	}

	query = `kube_job_created`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job", "creationTime", "Job", query)
	}

	//CronJob metrics
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Cron Job Metrics")
		fmt.Println("[DEBUG] message=Collecting Cron Job Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}
	query = `kube_cronjob_labels`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", "label", "CronJob", query)
	}

	query = `kube_cronjob_info`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobInfo query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", "label", "CronJob", query)
	}

	query = `kube_cronjob_next_schedule_time`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobNextScheduleTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobNextScheduleTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", query, "CronJob", query)
	}

	query = `kube_cronjob_status_last_schedule_time`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobStatusLastScheduleTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobStatusLastScheduleTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", query, "CronJob", query)
	}

	query = `kube_cronjob_status_active`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobStatusActive query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobStatusActive query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", query, "CronJob", query)
	}

	query = `kube_cronjob_created`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobCreated query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", "creationTime", "CronJob", query)
	}

	//HPA metrics
	if args.Debug {
		args.DebugLogger.Println("message=Collecting HPA Metrics")
		fmt.Println("[DEBUG] message=Collecting HPA Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}

	//Query to see what hpa labels may be there and if need to use newer or older versions of the query based on what we see for results. Note if you don't have any then will default to using newer version.
	query = `kube_hpa_labels`
	result, err = common.MetricCollect(args, query)
	var hpaName string
	var hpaLabel model.LabelName
	mat = result.(model.Matrix)
	n = mat.Len()
	if n != 0 {
		hpaName = "hpa"
		hpaLabel = "hpa"
	} else {
		hpaName = "horizontalpodautoscaler"
		hpaLabel = "horizontalpodautoscaler"
		query = `kube_` + hpaName + `_labels`
		result, err = common.MetricCollect(args, query)
	}

	if err != nil {
		args.WarnLogger.Println("metric=hpaLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=hpaLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, hpaLabel, "label", "Deployment", query)
	}

	if disc, err := args.ToDiscovery(common.ContainerEntityKind); err == nil {
		discovery := &datamodel.ContainerDiscovery{Discovery: disc, Namespaces: systems}
		common.WriteDiscovery(args, discovery, common.ContainerEntityKind)
	}

	//Container workloads
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Container Workload Metrics")
		fmt.Println("[DEBUG] message=Collecting Container Workload Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}

	query = `container_cpu_usage_seconds_total`
	common.GetWorkload("container_cpu_usage_seconds_total", query, args, common.ContainerEntityKind)

	query = `container_memory_usage_bytes`
	common.GetWorkload("container_memory_usage_bytes", query, args, common.ContainerEntityKind)

	query = `container_memory_rss`
	common.GetWorkload("container_memory_rss", query, args, common.ContainerEntityKind)

	query = `container_fs_usage_bytes`
	common.GetWorkload("container_fs_usage_bytes", query, args, common.ContainerEntityKind)

	query = `kube_pod_container_status_restarts_total`
	common.GetWorkload("kube_pod_container_status_restarts_total", query, args, common.ContainerEntityKind)

	if labelSuffix == "" {
		query = `kube_` + hpaName + `_status_condition{status="true",condition="ScalingLimited"}`
	} else {
		query = `kube_` + hpaName + `_status_condition{status="ScalingLimited",condition="true"}`
	}
	common.GetWorkload(`kube_`+hpaName+`_status_condition`, query, args, common.ContainerEntityKind)

	//HPA workloads
	query = `kube_` + hpaName + `_spec_max_replicas`
	common.GetWorkload(`kube_`+hpaName+`_spec_max_replicas`, query, args, common.ContainerEntityKind)

	query = `kube_` + hpaName + `_spec_min_replicas`
	common.GetWorkload(`kube_`+hpaName+`_spec_min_replicas`, query, args, common.ContainerEntityKind)

	query = `kube_` + hpaName + `_status_current_replicas`
	common.GetWorkload(`kube_`+hpaName+`_status_current_replicas`, query, args, common.ContainerEntityKind)
}

func newMidLevel() *datamodel.MidLevel {
	return &datamodel.MidLevel{
		OwnerName:          &datamodel.Labels{},
		OwnerKind:          &datamodel.Labels{},
		CreationTime:       &datamodel.Labels{},
		NextSchedTime:      &datamodel.Labels{},
		StatusActive:       &datamodel.Labels{},
		LastSchedTime:      &datamodel.Labels{},
		MetadataGeneration: &datamodel.Labels{},
		MaxSurge:           &datamodel.Labels{},
		MaxUnavailable:     &datamodel.Labels{},
		Completions:        &datamodel.Labels{},
		Parallelism:        &datamodel.Labels{},
		CompletionTime:     &datamodel.Labels{},
		Containers:         make(map[string]*datamodel.Container),
		LabelMap:           make(datamodel.LabelMap),
	}
}

func newContainer() *datamodel.Container {
	return &datamodel.Container{
		CpuLimit:   &datamodel.Labels{},
		CpuRequest: &datamodel.Labels{},
		MemLimit:   &datamodel.Labels{},
		MemRequest: &datamodel.Labels{},
		PowerState: &datamodel.Labels{},
		LabelMap:   make(datamodel.LabelMap),
	}
}

func getMidLevelOwner(args *common.Parameters, query, entityKey, metricKey string) error {
	var result model.Value
	var err error
	query += `{owner_name!="<none>"}`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		errMsg := fmt.Sprintf("metric=%s query=%s message=%v", strings.ToLower(entityKey), query, err)
		args.ErrorLogger.Println(errMsg)
		fmt.Printf("[ERROR] %s\n", errMsg)
		return err
	}
	mat := result.(model.Matrix)
	n := mat.Len()
	for i := 0; i < n; i++ {
		// Get the entity, namespace owner kind and owner kind
		entityName := string(mat[i].Metric[model.LabelName(metricKey)])
		namespaceName := string(mat[i].Metric[common.NamespaceKey])
		ownerKind := string(mat[i].Metric[ownerKindKey])
		ownerName := string(mat[i].Metric[ownerNameKey])
		if entityName == "" || namespaceName == "" || ownerKind == "" || ownerName == "" {
			continue
		}
		var entity *datamodel.MidLevel
		var ok bool
		// If we have an owner but haven't seen the entity - skip it
		if entity, ok = systems[namespaceName].Entities[entityKey][entityName]; !ok {
			continue
		}

		_ = entity.OwnerKind.AppendSampleStreamWithFilter(mat[i], ownerKindFilter)
		_ = entity.OwnerName.AppendSampleStreamWithFilter(mat[i], ownerNameFilter)
		// Create the entity if doesn't exist.
		if _, ok := systems[namespaceName].Entities[ownerKind]; !ok {
			systems[namespaceName].Entities[ownerKind] = make(map[string]*datamodel.MidLevel)
		}
		if _, ok := systems[namespaceName].Entities[ownerKind][ownerName]; !ok {
			systems[namespaceName].Entities[ownerKind][ownerName] = newMidLevel()
		}
	}
	return nil
}
