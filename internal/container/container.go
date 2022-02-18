//Package container2 collects data related to containers and formats into json files to send to Densify.
package container

import (
	"fmt"
	"runtime"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

var systems = map[string]*datamodel.Namespace{}
var entityKind = "container"

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
		namespaceValue, ok := mat[i].Metric["namespace"]
		if !ok {
			continue
		}
		if _, ok = systems[string(namespaceValue)]; !ok {
			continue
		}
		//Validate that the data contains the pod label with value and check it exists in our systems structure
		podValue, ok := mat[i].Metric[pod]
		if !ok {
			continue
		}
		if _, ok = systems[string(namespaceValue)].Entities["Pods"][string(podValue)]; !ok {
			continue
		}
		//Validate that the data contains the container label with value and check it exists in our systems structure
		containerNameValue, ok := mat[i].Metric[container]
		if !ok {
			continue
		}
		var container *datamodel.Container
		if container, ok = systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerNameValue)]; !ok {
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
			_ = container.PowerState.AppendSampleStreamWithValue(mat[i], "", nil)
		default:
			var labels *datamodel.Labels
			if labels, ok = container.LabelMap[metric]; !ok {
				labels = &datamodel.Labels{}
				container.LabelMap[metric] = labels
			}
			_ = labels.AppendSampleStream(mat[i])
		}
	}
	return true
}

func cpuConvert(value float64) float64 {
	return value * 1000
}

func memConvert(value float64) float64 {
	return value / 1024 / 1024
}

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
		namespaceValue, ok := mat[i].Metric["namespace"]
		if !ok {
			continue
		}
		if _, ok = systems[string(namespaceValue)]; !ok {
			continue
		}
		//Validate that the data contains the mid label with value and check it exists in our systems structure
		midValue, ok := mat[i].Metric[mid]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)].Entities[kind][string(midValue)]; !ok {
			continue
		}

		//validates that the value of the entity is set and if not will default to 0
		var value int64
		if len(mat[i].Values) == 0 {
			value = 0
		} else {
			value = int64(mat[i].Values[len(mat[i].Values)-1].Value)
		}
		//Check which metric this is for and update the corresponding variable for this mid in the system data structure
		switch metric {
		case "label":
			var labels *datamodel.Labels
			if labels, ok = systems[string(namespaceValue)].Entities[kind][string(midValue)].LabelMap[query]; !ok {
				labels = &datamodel.Labels{}
				systems[string(namespaceValue)].Entities[kind][string(midValue)].LabelMap[query] = labels
			}
			_ = labels.AppendSampleStream(mat[i])
		case "creationTime":
			systems[string(namespaceValue)].Entities[kind][string(midValue)].CreationTime = value
		case "kube_cronjob_next_schedule_time":
			systems[string(namespaceValue)].Entities[kind][string(midValue)].NextSchedTime = value
		case "kube_cronjob_status_active":
			systems[string(namespaceValue)].Entities[kind][string(midValue)].StatusActive = value
		case "kube_cronjob_status_last_schedule_time":
			systems[string(namespaceValue)].Entities[kind][string(midValue)].LastSchedTime = value
		case "kube_deployment_metadata_generation":
			systems[string(namespaceValue)].Entities[kind][string(midValue)].MetadataGeneration = value
		case "kube_deployment_spec_strategy_rollingupdate_max_surge":
			systems[string(namespaceValue)].Entities[kind][string(midValue)].MaxSurge = value
		case "kube_deployment_spec_strategy_rollingupdate_max_unavailable":
			systems[string(namespaceValue)].Entities[kind][string(midValue)].MaxUnavailable = value
		case "kube_job_status_completion_time":
			systems[string(namespaceValue)].Entities[kind][string(midValue)].CompletionTime = value
		case "kube_job_spec_completions":
			systems[string(namespaceValue)].Entities[kind][string(midValue)].Completions = value
		case "kube_job_spec_parallelism":
			systems[string(namespaceValue)].Entities[kind][string(midValue)].Parallelism = value
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
		namespaceValue, ok := mat[i].Metric["namespace"]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; !ok {
			continue
		}
		labels := &datamodel.Labels{}
		systems[string(namespaceValue)].LabelMap[query] = labels
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
			podName := string(mat[i].Metric["pod"])
			namespaceName := string(mat[i].Metric["namespace"])

			//check if already have setup namespace, pod in system structure and if not add them.
			if _, ok := systems[namespaceName]; !ok {
				systems[namespaceName] = &datamodel.Namespace{LabelMap: make(datamodel.LabelMap), Entities: map[string]map[string]*datamodel.MidLevel{}}
				systems[namespaceName].Entities["Pods"] = map[string]*datamodel.MidLevel{}
			}
			if _, ok := systems[namespaceName].Entities["Pods"][podName]; !ok {
				systems[namespaceName].Entities["Pods"][podName] = &datamodel.MidLevel{Containers: map[string]*datamodel.Container{}, LabelMap: make(datamodel.LabelMap)}
			}
			//If the container doesn't exist then look to add and call getContainerMetric to set labels.
			if _, ok := systems[namespaceName].Entities["Pods"][podName].Containers[containerName]; !ok {
				systems[namespaceName].Entities["Pods"][podName].Containers[containerName] = &datamodel.Container{
					CpuLimit:   &datamodel.Labels{},
					CpuRequest: &datamodel.Labels{},
					MemLimit:   &datamodel.Labels{},
					MemRequest: &datamodel.Labels{},
					PowerState: &datamodel.Labels{},
					LabelMap:   make(datamodel.LabelMap),
				}
				getContainerMetric(result, "pod", "container", query)
			}
		}
	}

	//Create any entities based on pods that are owned by them. This will create replicasets, jobs, daemonsets, etc.
	query = `kube_pod_owner{owner_name!="<none>"}`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.ErrorLogger.Println("metric=pods query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=pods query=" + query + " message=" + err.Error())
		return
	} else {
		mat := result.(model.Matrix)
		n := mat.Len()
		for i := 0; i < n; i++ {
			//Get the pod, and namespace names as well as what the owner name and kind is.
			podName := string(mat[i].Metric["pod"])
			namespaceName := string(mat[i].Metric["namespace"])
			ownerKind := string(mat[i].Metric["owner_kind"])
			ownerName := string(mat[i].Metric["owner_name"])
			if podName == "" || namespaceName == "" || ownerKind == "" || ownerName == "" {
				continue
			}

			//If we have an owner but haven't seen the pod we skip it as currently metrics for analysis is based on pod\container so having info for non-existant pod isn't of use.
			if _, ok := systems[namespaceName].Entities["Pods"][podName]; !ok {
				continue
			}

			//For the pod set who its owner is for mapping later.
			systems[namespaceName].Entities["Pods"][podName].OwnerKind = ownerKind
			systems[namespaceName].Entities["Pods"][podName].OwnerName = ownerName
			//Create the entity if doesn't exist.
			if _, ok := systems[namespaceName].Entities[ownerKind]; !ok {
				systems[namespaceName].Entities[ownerKind] = map[string]*datamodel.MidLevel{}
			}
			if _, ok := systems[namespaceName].Entities[ownerKind][ownerName]; !ok {
				systems[namespaceName].Entities[ownerKind][ownerName] = &datamodel.MidLevel{LabelMap: make(datamodel.LabelMap), OwnerName: "", OwnerKind: ""}
			}
		}
	}

	//Check for any replicaset that are owned by deployments.
	query = `kube_replicaset_owner{owner_name!="<none>"}`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=replicasets query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicasets query=" + query + " message=" + err.Error())
	} else {
		mat := result.(model.Matrix)
		n := mat.Len()
		for i := 0; i < n; i++ {
			replicaSetName := string(mat[i].Metric["replicaset"])
			namespaceName := string(mat[i].Metric["namespace"])
			ownerKind := string(mat[i].Metric["owner_kind"])
			ownerName := string(mat[i].Metric["owner_name"])
			if replicaSetName == "" || namespaceName == "" || ownerKind == "" || ownerName == "" {
				continue
			}

			if _, ok := systems[namespaceName].Entities["ReplicaSet"][replicaSetName]; !ok {
				continue
			}

			//Update the replicaset with info about the deployment that owns it.
			systems[namespaceName].Entities["ReplicaSet"][replicaSetName].OwnerKind = ownerKind
			systems[namespaceName].Entities["ReplicaSet"][replicaSetName].OwnerName = ownerName

			//Check if the deployment exists and if not add it.
			if _, ok := systems[namespaceName].Entities[ownerKind]; !ok {
				systems[namespaceName].Entities[ownerKind] = map[string]*datamodel.MidLevel{}
			}

			if _, ok := systems[namespaceName].Entities[ownerKind][ownerName]; !ok {
				systems[namespaceName].Entities[ownerKind][ownerName] = &datamodel.MidLevel{LabelMap: make(datamodel.LabelMap), OwnerName: "", OwnerKind: ""}
			}
		}
	}

	//Check for any jobs that are owned by Cronjobs
	query = `kube_job_owner{owner_name!="<none>"}`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=jobs query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobs query=" + query + " message=" + err.Error())
	} else {
		mat := result.(model.Matrix)
		n := mat.Len()
		for i := 0; i < n; i++ {
			jobName := string(mat[i].Metric["job_name"])
			namespaceName := string(mat[i].Metric["namespace"])
			ownerKind := string(mat[i].Metric["owner_kind"])
			ownerName := string(mat[i].Metric["owner_name"])
			if jobName == "" || namespaceName == "" || ownerKind == "" || ownerName == "" {
				continue
			}

			if _, ok := systems[namespaceName].Entities["Job"][jobName]; !ok {
				continue
			}

			//Set the owner of the job to be the name of the cronjob.
			systems[namespaceName].Entities["Job"][jobName].OwnerKind = ownerKind
			systems[namespaceName].Entities["Job"][jobName].OwnerName = ownerName

			//Check if the Cronjob exists and if not add it
			if _, ok := systems[namespaceName].Entities[ownerKind]; !ok {
				systems[namespaceName].Entities[ownerKind] = map[string]*datamodel.MidLevel{}
			}
			if _, ok := systems[namespaceName].Entities[ownerKind][ownerName]; !ok {
				systems[namespaceName].Entities[ownerKind][ownerName] = &datamodel.MidLevel{LabelMap: make(datamodel.LabelMap), OwnerName: "", OwnerKind: ""}
			}
		}
	}

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

	//Getting info from cAdvisor metric that will be stored as labels.
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
		getMidMetric(result, "pod", "label", "Pods", query)
	}

	query = `kube_pod_labels`
	result, err = common.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=podLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "pod", "label", "Pods", query)
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

	var cluster = map[string]*datamodel.ContainerCluster{}
	cluster["cluster"] = &datamodel.ContainerCluster{Namespaces: systems, Name: *args.ClusterName}
	common.WriteDiscovery(args, cluster, entityKind)

	//Container workloads
	if args.Debug {
		args.DebugLogger.Println("message=Collecting Container Workload Metrics")
		fmt.Println("[DEBUG] message=Collecting Container Workload Metrics")
		runtime.ReadMemStats(&mem)
		args.DebugLogger.Printf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
		fmt.Printf("[DEBUG] Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", mem.Alloc/1024/1024, mem.TotalAlloc/1024/1024, mem.Sys/1024/1024, mem.NumGC)
	}

	query = `container_cpu_usage_seconds_total`
	common.GetWorkload("container_cpu_usage_seconds_total", query, args, entityKind)

	query = `container_memory_usage_bytes`
	common.GetWorkload("container_memory_usage_bytes", query, args, entityKind)

	query = `container_memory_rss`
	common.GetWorkload("container_memory_rss", query, args, entityKind)

	query = `container_fs_usage_bytes`
	common.GetWorkload("container_fs_usage_bytes", query, args, entityKind)

	query = `kube_pod_container_status_restarts_total`
	common.GetWorkload("kube_pod_container_status_restarts_total", query, args, entityKind)

	if labelSuffix == "" {
		query = `kube_` + hpaName + `_status_condition{status="true",condition="ScalingLimited"}`
	} else {
		query = `kube_` + hpaName + `_status_condition{status="ScalingLimited",condition="true"}`
	}
	common.GetWorkload(`kube_`+hpaName+`_status_condition`, query, args, entityKind)

	//HPA workloads
	query = `kube_` + hpaName + `_spec_max_replicas`
	common.GetWorkload(`kube_`+hpaName+`_spec_max_replicas`, query, args, entityKind)

	query = `kube_` + hpaName + `_spec_min_replicas`
	common.GetWorkload(`kube_`+hpaName+`_spec_min_replicas`, query, args, entityKind)

	query = `kube_` + hpaName + `_status_current_replicas`
	common.GetWorkload(`kube_`+hpaName+`_status_current_replicas`, query, args, entityKind)
}
