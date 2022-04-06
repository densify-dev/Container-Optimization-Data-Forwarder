// Package container collects data related to containers and formats into json files to send to Densify.
package container

import (
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
	"runtime"
	"strings"
)

var systems = map[string]*datamodel.Namespace{}

const (
	ownerKindKey        = "owner_kind"
	ownerNameKey        = "owner_name"
	podMetricKey        = "pod"
	replicaSetMetricKey = "replicaset"
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
		nsValue, ok := mat[i].Metric[datamodel.NamespaceMetricKey]
		if !ok {
			continue
		}
		namespaceValue := string(nsValue)
		//Validate that the data contains the pod label with value and check it exists in our systems structure
		pValue, ok := mat[i].Metric[pod]
		if !ok {
			continue
		}
		podValue := string(pValue)
		var ei datamodel.EntityInterface
		if ei, ok = datamodel.GetEntity(systems, namespaceValue, datamodel.PodKey, podValue); !ok || ei == nil {
			continue
		}
		var p *datamodel.Pod
		if p, ok = ei.(*datamodel.Pod); !ok {
			continue
		}
		cValue, ok := mat[i].Metric[container]
		if !ok {
			continue
		}
		contValue := string(cValue)
		var container *datamodel.Container
		if container, ok = datamodel.GetContainer(p, contValue); !ok {
			continue
		}

		//Check which metric this is for and update the corresponding variable for this container in the system data structure
		switch metric {
		case "limits":
			switch mat[i].Metric["resource"] {
			case "memory":
				_ = container.MemLimit.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, memConvert)
			case "cpu":
				_ = container.CpuLimit.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, cpuConvert)
			}
		case "requests":
			switch mat[i].Metric["resource"] {
			case "memory":
				_ = container.MemRequest.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, memConvert)
			case "cpu":
				_ = container.CpuRequest.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, cpuConvert)
			}
		case "cpuLimit":
			_ = container.CpuLimit.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, cpuConvert)
		case "cpuRequest":
			_ = container.CpuRequest.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, cpuConvert)
		case "memLimit":
			_ = container.MemLimit.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, memConvert)
		case "memRequest":
			_ = container.MemRequest.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, memConvert)
		case "powerState":
			_ = container.PowerState.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, boolConv)
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

func getEntityMetric(result model.Value, mid model.LabelName, metric, kind, query string) {
	mat, ok := result.(model.Matrix)
	if !ok {
		return
	}
	// Loop through the different entities in the results.
	n := mat.Len()
	for i := 0; i < n; i++ {
		//Validate that the data contains the namespace label with value and check it exists in our systems structure.
		nsVal, ok := mat[i].Metric[datamodel.NamespaceMetricKey]
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
		var ei datamodel.EntityInterface
		if ei, ok = datamodel.GetEntity(systems, namespaceValue, kind, midValue); !ok || ei == nil {
			continue
		}
		switch metric {
		case "label":
			labels := datamodel.EnsureLabels(ei.Get().LabelMap, query)
			_ = labels.AppendSampleStream(mat[i])
		case "creationTime":
			_ = ei.Get().CreationTime.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, timeConv)
		case "kube_cronjob_next_schedule_time":
			if cj, ok := ei.(*datamodel.CronJob); ok {
				_ = cj.NextScheduledTime.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, timeConv)
			}
		case "kube_cronjob_status_active":
			if cj, ok := ei.(*datamodel.CronJob); ok {
				_ = cj.Active.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, boolConv)
			}
		case "kube_cronjob_status_last_schedule_time":
			if cj, ok := ei.(*datamodel.CronJob); ok {
				_ = cj.LastScheduledTime.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, timeConv)
			}
		case "kube_deployment_metadata_generation":
			_ = ei.Get().Generation.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, nil)
		case "kube_deployment_spec_strategy_rollingupdate_max_surge":
			if d, ok := ei.(*datamodel.Deployment); ok {
				_ = d.MaxSurge.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, nil)
			}
		case "kube_deployment_spec_strategy_rollingupdate_max_unavailable":
			if d, ok := ei.(*datamodel.Deployment); ok {
				_ = d.MaxUnavailable.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, nil)
			}
		case "kube_job_status_completion_time":
			if j, ok := ei.(*datamodel.Job); ok {
				_ = j.CompletionTime.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, timeConv)
			}
		case "kube_job_spec_completions":
			if j, ok := ei.(*datamodel.Job); ok {
				_ = j.Completions.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, nil)
			}
		case "kube_job_spec_parallelism":
			if j, ok := ei.(*datamodel.Job); ok {
				_ = j.Parallelism.AppendSampleStreamWithValue(mat[i], datamodel.SingleValueKey, nil)
			}
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
		nsValue, ok := mat[i].Metric[datamodel.NamespaceMetricKey]
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
func Metrics(args *prometheus.Parameters) {
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
	result, err = prometheus.MetricCollect(args, query)
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
			namespaceName := string(mat[i].Metric[datamodel.NamespaceMetricKey])
			// ensure the namespace and pods are already there
			_, _ = datamodel.EnsureEntity(systems, namespaceName, datamodel.NamespaceKey, "")
			ei, _ := datamodel.EnsureEntity(systems, namespaceName, datamodel.PodKey, podName)
			_, _ = datamodel.EnsureContainer(ei.(*datamodel.Pod), containerName)
			getContainerMetric(result, "pod", "container", query)
		}
	}

	// Create any entities based on pods that are owned by them.
	// This will create replicasets, jobs, daemonsets, etc.
	if err = getEntityOwner(args, "kube_pod_owner", datamodel.PodKey, podMetricKey); err != nil {
		return
	}

	// Check for any replicaset that are owned by deployments.
	_ = getEntityOwner(args, "kube_replicaset_owner", datamodel.ReplicaSetKey, replicaSetMetricKey)

	//Check for any jobs that are owned by Cronjobs
	_ = getEntityOwner(args, "kube_job_owner", datamodel.JobKey, jobMetricKey)

	if args.Debug {
		args.DebugLogger.Println("message=Collecting Container Metrics")
		fmt.Println("[DEBUG] message=Collecting Container Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}

	//query for the Container resource limits (cpu and memory) if we get no results for the newer combined metric then will fall back to older metrics which get just CPU and Memory.
	query = `kube_pod_container_resource_limits`
	result, err = prometheus.MetricCollect(args, query)
	mat := result.(model.Matrix)
	n := mat.Len()
	if n == 0 {
		query = `kube_pod_container_resource_limits_cpu_cores`
		result, err = prometheus.MetricCollect(args, query)
		if err != nil {
			args.WarnLogger.Println("metric=cpuLimit query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cpuLimit query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "cpuLimit")
		}

		query = `kube_pod_container_resource_limits_memory_bytes`
		result, err = prometheus.MetricCollect(args, query)
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
	result, err = prometheus.MetricCollect(args, query)
	mat = result.(model.Matrix)
	n = mat.Len()
	if n == 0 {
		query = `kube_pod_container_resource_requests_cpu_cores`
		result, err = prometheus.MetricCollect(args, query)
		if err != nil {
			args.WarnLogger.Println("metric=cpuRequest query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cpuRequest query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "cpuRequest")
		}

		query = `kube_pod_container_resource_requests_memory_bytes`
		result, err = prometheus.MetricCollect(args, query)
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
	result, err = prometheus.MetricCollect(args, query)
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
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=podInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podInfo query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "pod", "label", datamodel.PodKey, query)
	}

	query = `kube_pod_labels`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=podLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podLabels query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "pod", "label", datamodel.PodKey, query)
	}

	//Depending on version of KSM will determine what metric need to use to see if containers are terminated or not for there state.
	query = `kube_pod_container_status_terminated`
	result, err = prometheus.MetricCollect(args, query)
	mat = result.(model.Matrix)
	n = mat.Len()
	if n == 0 {
		query = `kube_pod_container_status_terminated_reason`
		result, err = prometheus.MetricCollect(args, query)
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
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=podCreationTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podCreationTime query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "pod", "creationTime", "Pod", query)
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
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=namespaceLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=namespaceLabels query=" + query + " message=" + err.Error())
	} else {
		getNamespaceMetric(result, query)
	}

	query = `kube_namespace_annotations`
	result, err = prometheus.MetricCollect(args, query)
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
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=labels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=labels query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "deployment", "label", "Deployment", query)
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_surge`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=maxSurge query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=maxSurge query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "deployment", query, "Deployment", query)
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_unavailable`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=maxUnavailable query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=maxUnavailable query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "deployment", query, "Deployment", query)
	}

	query = `kube_deployment_metadata_generation`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=metadataGeneration query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=metadataGeneration query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "deployment", query, "Deployment", query)
	}

	query = `kube_deployment_created`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=deploymentCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=deploymentCreated query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "deployment", "creationTime", "Deployment", query)
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
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=replicaSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicaSetLabels query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "replicaset", "label", "ReplicaSet", query)
	}

	query = `kube_replicaset_created`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=replicaSetCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicaSetCreated query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "replicaset", "creationTime", "ReplicaSet", query)
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
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=replicationControllerCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicationControllerCreated query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "replicationcontroller", "creationTime", "ReplicationController", query)
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
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=daemonSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=daemonSetLabels query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "daemonset", "label", "DaemonSet", query)
	}

	query = `kube_daemonset_created`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=daemonSetCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=daemonSetCreated query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "daemonset", "creationTime", "DaemonSet", query)
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
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=statefulSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=statefulSetLabels query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "statefulset", "label", "StatefulSet", query)
	}

	query = `kube_statefulset_created`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=statefulSetCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=statefulSetCreated query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "statefulset", "creationTime", "StatefulSet", query)
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
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobInfo query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "job_name", "label", "Job", query)
	}

	query = `kube_job_labels`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobLabel query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobLabel query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "job_name", "label", "Job", query)
	}

	query = `kube_job_spec_completions`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobSpecCompletions query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobSpecCompletions query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "job_name", query, "Job", query)
	}

	query = `kube_job_spec_parallelism`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobSpecParallelism query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobSpecParallelism query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "job_name", query, "Job", query)
	}

	query = `kube_job_status_completion_time`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobStatusCompletionTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobStatusCompletionTime query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "job_name", query, "Job", query)
	}

	query = `kube_job_status_start_time`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobStatusStartTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobStatusStartTime query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "job_name", "creationTime", "Job", query)
	}

	query = `kube_job_created`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobCreated query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "job", "creationTime", "Job", query)
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
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobLabels query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "cronjob", "label", "CronJob", query)
	}

	query = `kube_cronjob_info`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobInfo query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "cronjob", "label", "CronJob", query)
	}

	query = `kube_cronjob_next_schedule_time`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobNextScheduleTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobNextScheduleTime query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "cronjob", query, "CronJob", query)
	}

	query = `kube_cronjob_status_last_schedule_time`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobStatusLastScheduleTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobStatusLastScheduleTime query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "cronjob", query, "CronJob", query)
	}

	query = `kube_cronjob_status_active`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobStatusActive query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobStatusActive query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "cronjob", query, "CronJob", query)
	}

	query = `kube_cronjob_created`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=cronJobCreated query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobCreated query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, "cronjob", "creationTime", "CronJob", query)
	}

	//HPA metrics

	// TODO: HPA is not ONLY for deployments - should also implement for RCs, StatefulSets, ... ?
	if args.Debug {
		args.DebugLogger.Println("message=Collecting HPA Metrics")
		fmt.Println("[DEBUG] message=Collecting HPA Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}

	//Query to see what hpa labels may be there and if need to use newer or older versions of the query based on what we see for results. Note if you don't have any then will default to using newer version.
	query = `kube_hpa_labels`
	result, err = prometheus.MetricCollect(args, query)
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
		result, err = prometheus.MetricCollect(args, query)
	}

	if err != nil {
		args.WarnLogger.Println("metric=hpaLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=hpaLabels query=" + query + " message=" + err.Error())
	} else {
		getEntityMetric(result, hpaLabel, "label", "Deployment", query)
	}

	if disc, err := args.ToDiscovery(prometheus.ContainerEntityKind); err == nil {
		discovery := &datamodel.ContainerDiscovery{Discovery: disc, Namespaces: systems}
		prometheus.WriteDiscovery(args, discovery, prometheus.ContainerEntityKind)
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
	prometheus.GetWorkload("container_cpu_usage_seconds_total", query, args, prometheus.ContainerEntityKind)

	query = `container_memory_usage_bytes`
	prometheus.GetWorkload("container_memory_usage_bytes", query, args, prometheus.ContainerEntityKind)

	query = `container_memory_rss`
	prometheus.GetWorkload("container_memory_rss", query, args, prometheus.ContainerEntityKind)

	query = `container_fs_usage_bytes`
	prometheus.GetWorkload("container_fs_usage_bytes", query, args, prometheus.ContainerEntityKind)

	query = `kube_pod_container_status_restarts_total`
	prometheus.GetWorkload("kube_pod_container_status_restarts_total", query, args, prometheus.ContainerEntityKind)

	//HPA
	// event
	if labelSuffix == "" {
		query = `kube_` + hpaName + `_status_condition{status="true",condition="ScalingLimited"}`
	} else {
		query = `kube_` + hpaName + `_status_condition{status="ScalingLimited",condition="true"}`
	}
	prometheus.GetWorkload(`kube_`+hpaName+`_status_condition`, query, args, prometheus.ContainerEntityKind)

	// discovery
	query = `kube_` + hpaName + `_spec_max_replicas`
	prometheus.GetWorkload(`kube_`+hpaName+`_spec_max_replicas`, query, args, prometheus.ContainerEntityKind)

	// discovery
	query = `kube_` + hpaName + `_spec_min_replicas`
	prometheus.GetWorkload(`kube_`+hpaName+`_spec_min_replicas`, query, args, prometheus.ContainerEntityKind)

	// event
	query = `kube_` + hpaName + `_status_current_replicas`
	prometheus.GetWorkload(`kube_`+hpaName+`_status_current_replicas`, query, args, prometheus.ContainerEntityKind)
}

func getEntityOwner(args *prometheus.Parameters, query, entityKey, metricKey string) error {
	var result model.Value
	var err error
	query += `{owner_name!="<none>"}`
	result, err = prometheus.MetricCollect(args, query)
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
		namespaceName := string(mat[i].Metric[datamodel.NamespaceMetricKey])
		ownerKind := string(mat[i].Metric[ownerKindKey])
		ownerName := string(mat[i].Metric[ownerNameKey])
		if entityName == "" || namespaceName == "" || ownerKind == "" || ownerName == "" {
			continue
		}
		var ei datamodel.EntityInterface
		var ok bool
		// If we have an owner but haven't seen the entity - skip it
		if ei, ok = datamodel.GetEntity(systems, namespaceName, entityKey, entityName); !ok {
			continue
		}
		entity := ei.Get()

		_ = entity.OwnerKind.AppendSampleStreamWithFilter(mat[i], ownerKindFilter)
		_ = entity.OwnerName.AppendSampleStreamWithFilter(mat[i], ownerNameFilter)
		// Create the owner entity if it doesn't exist.
		_, _ = datamodel.EnsureEntity(systems, namespaceName, ownerKind, ownerName)
	}
	return nil
}
