//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"runtime"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

var systems = map[string]*datamodel.Namespace{}
var entityKind = "container"

//getContainerMetric is used to parse the results from Prometheus related to Container Entities and store them in the systems data structure.
func getContainerMetric(result model.Value, pod, container model.LabelName, metric string) bool {
	var status = false
	//Validate there is data in the results.
	if result == nil {
		return status
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Validate that the data contains the namespace label with value and check it exists in our systems structure.
		namespaceValue, test := result.(model.Matrix)[i].Metric["namespace"]
		if !test {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; !ok {
			continue
		}
		//Validate that the data contains the pod label with value and check it exists in our systems structure
		podValue, test := result.(model.Matrix)[i].Metric[pod]
		if !test {
			continue
		}
		if _, ok := systems[string(namespaceValue)].Entities["Pods"][string(podValue)]; !ok {
			continue
		}
		//Validate that the data contains the container label with value and check it exists in our systems structure
		containerValue, test := result.(model.Matrix)[i].Metric[container]
		if !test {
			continue
		}
		if _, ok := systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)]; !ok {
			continue
		}

		//validates that the value of the entity is set and if not will default to 0
		var value float64
		if len(result.(model.Matrix)[i].Values) == 0 {
			value = 0
		} else {
			value = float64(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
		}

		//Check which metric this is for and update the corresponding variable for this container in the system data structure
		switch metric {
		case "limits":
			switch result.(model.Matrix)[i].Metric["resource"] {
			case "memory":
				systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].MemLimit = int(value / 1024 / 1024)
			case "cpu":
				systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].CpuLimit = int(value * 1000)
			}
		case "requests":
			switch result.(model.Matrix)[i].Metric["resource"] {
			case "memory":
				systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].MemRequest = int(value / 1024 / 1024)
			case "cpu":
				systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].CpuRequest = int(value * 1000)
			}
		case "cpuLimit":
			systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].CpuLimit = int(value)
		case "cpuRequest":
			systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].CpuRequest = int(value)
		case "memLimit":
			systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].MemLimit = int(value)
		case "memRequest":
			systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].MemRequest = int(value)
		case "powerState":
			systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].PowerState = int(value)
		default:
			if _, ok := systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].LabelMap[metric]; !ok {
				systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].LabelMap[metric] = map[string]string{}
			}
			//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
			for key, value := range result.(model.Matrix)[i].Metric {
				common.AddToLabelMap(string(key), string(value), systems[string(namespaceValue)].Entities["Pods"][string(podValue)].Containers[string(containerValue)].LabelMap[metric])
			}
		}
		status = true
	}

	return status
}

//getmidMetric is used to parse the results from Prometheus related to mid Entities and store them in the systems data structure.
func getMidMetric(result model.Value, mid model.LabelName, metric, kind, query string) {
	//Validate there is data in the results.
	if result == nil {
		return
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Validate that the data contains the namespace label with value and check it exists in our systems structure.
		namespaceValue, ok := result.(model.Matrix)[i].Metric["namespace"]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; !ok {
			continue
		}
		//Validate that the data contains the mid label with value and check it exists in our systems structure
		midValue, ok := result.(model.Matrix)[i].Metric[mid]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)].Entities[kind][string(midValue)]; !ok {
			continue
		}

		//validates that the value of the entity is set and if not will default to 0
		var value int64
		if len(result.(model.Matrix)[i].Values) == 0 {
			value = 0
		} else {
			value = int64(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
		}
		//Check which metric this is for and update the corresponding variable for this mid in the system data structure
		switch metric {
		case "label":
			if _, ok := systems[string(namespaceValue)].Entities[kind][string(midValue)].LabelMap[query]; !ok {
				systems[string(namespaceValue)].Entities[kind][string(midValue)].LabelMap[query] = map[string]string{}
			}
			//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
			for key, value := range result.(model.Matrix)[i].Metric {
				common.AddToLabelMap(string(key), string(value), systems[string(namespaceValue)].Entities[kind][string(midValue)].LabelMap[query])
			}
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
	//Validate there is data in the results.
	if result == nil {
		return
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
		namespaceValue, ok := result.(model.Matrix)[i].Metric["namespace"]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; !ok {
			continue
		}
		systems[string(namespaceValue)].LabelMap[query] = map[string]string{}
		//loop through all the labels for an entity and store them in a map.
		for key, value := range result.(model.Matrix)[i].Metric {
			common.AddToLabelMap(string(key), string(value), systems[string(namespaceValue)].LabelMap[query])
		}
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

	query = `kube_pod_container_info`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.ErrorLogger.Println("metric=containers query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=containers query=" + query + " message=" + err.Error())
		return
	} else {
		//Add containers to pods structure
		for i := 0; i < result.(model.Matrix).Len(); i++ {

			containerName := string(result.(model.Matrix)[i].Metric["container"])
			podName := string(result.(model.Matrix)[i].Metric["pod"])
			namespaceName := string(result.(model.Matrix)[i].Metric["namespace"])

			if _, ok := systems[namespaceName]; !ok {
				systems[namespaceName] = &datamodel.Namespace{LabelMap: map[string]map[string]string{}, Entities: map[string]map[string]*datamodel.MidLevel{}}
				systems[namespaceName].Entities["Pods"] = map[string]*datamodel.MidLevel{}
			}
			if _, ok := systems[namespaceName].Entities["Pods"][podName]; !ok {
				systems[namespaceName].Entities["Pods"][podName] = &datamodel.MidLevel{Containers: map[string]*datamodel.Container{}, LabelMap: map[string]map[string]string{}}
			}
			if _, ok := systems[namespaceName].Entities["Pods"][podName].Containers[containerName]; !ok {
				systems[namespaceName].Entities["Pods"][podName].Containers[containerName] = &datamodel.Container{LabelMap: map[string]map[string]string{}}
				getContainerMetric(result, "pod", "container", query)
			}
		}
	}

	query = `sum(kube_pod_owner{owner_name!="<none>"}) by (namespace, pod, owner_name, owner_kind)`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.ErrorLogger.Println("metric=pods query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=pods query=" + query + " message=" + err.Error())
		return
	} else {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			podName := string(result.(model.Matrix)[i].Metric["pod"])
			namespaceName := string(result.(model.Matrix)[i].Metric["namespace"])
			ownerKind := string(result.(model.Matrix)[i].Metric["owner_kind"])
			ownerName := string(result.(model.Matrix)[i].Metric["owner_name"])

			if _, ok := systems[namespaceName].Entities["Pods"][podName]; !ok {
				continue
			}

			systems[namespaceName].Entities["Pods"][podName].OwnerKind = ownerKind
			systems[namespaceName].Entities["Pods"][podName].OwnerName = ownerName
			if _, ok := systems[namespaceName].Entities[ownerKind]; !ok {
				systems[namespaceName].Entities[ownerKind] = map[string]*datamodel.MidLevel{}
			}
			if _, ok := systems[namespaceName].Entities[ownerKind][ownerName]; !ok {
				systems[namespaceName].Entities[ownerKind][ownerName] = &datamodel.MidLevel{LabelMap: map[string]map[string]string{}, OwnerName: "", OwnerKind: ""}
			}
		}
	}

	query = `sum(kube_replicaset_owner{owner_name!="<none>"}) by (namespace, replicaset, owner_name, owner_kind)`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=replicasets query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicasets query=" + query + " message=" + err.Error())
	} else {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			replicaSetName := string(result.(model.Matrix)[i].Metric["replicaset"])
			namespaceName := string(result.(model.Matrix)[i].Metric["namespace"])
			ownerKind := string(result.(model.Matrix)[i].Metric["owner_kind"])
			ownerName := string(result.(model.Matrix)[i].Metric["owner_name"])

			if _, ok := systems[namespaceName].Entities["ReplicaSet"][replicaSetName]; !ok {
				continue
			}

			systems[namespaceName].Entities["ReplicaSet"][replicaSetName].OwnerKind = ownerKind
			systems[namespaceName].Entities["ReplicaSet"][replicaSetName].OwnerName = ownerName

			if _, ok := systems[namespaceName].Entities[ownerKind]; !ok {
				systems[namespaceName].Entities[ownerKind] = map[string]*datamodel.MidLevel{}
			}

			if _, ok := systems[namespaceName].Entities[ownerKind][ownerName]; !ok {
				systems[namespaceName].Entities[ownerKind][ownerName] = &datamodel.MidLevel{LabelMap: map[string]map[string]string{}, OwnerName: "", OwnerKind: ""}
			}
		}
	}

	query = `sum(kube_job_owner{owner_name!="<none>"}) by (namespace, job_name, owner_name, owner_kind)`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=jobs query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobs query=" + query + " message=" + err.Error())
	} else {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			jobName := string(result.(model.Matrix)[i].Metric["job_name"])
			namespaceName := string(result.(model.Matrix)[i].Metric["namespace"])
			ownerKind := string(result.(model.Matrix)[i].Metric["owner_kind"])
			ownerName := string(result.(model.Matrix)[i].Metric["owner_name"])

			if _, ok := systems[namespaceName].Entities["Job"][jobName]; !ok {
				continue
			}

			systems[namespaceName].Entities["Job"][jobName].OwnerKind = ownerKind
			systems[namespaceName].Entities["Job"][jobName].OwnerName = ownerName

			if _, ok := systems[namespaceName].Entities[ownerKind]; !ok {
				systems[namespaceName].Entities[ownerKind] = map[string]*datamodel.MidLevel{}
			}
			if _, ok := systems[namespaceName].Entities[ownerKind][ownerName]; !ok {
				systems[namespaceName].Entities[ownerKind][ownerName] = &datamodel.MidLevel{LabelMap: map[string]map[string]string{}, OwnerName: "", OwnerKind: ""}
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

	query = `sum(kube_pod_container_resource_limits) by (pod,namespace,container,resource)`
	result, err = common.MetricCollect(args, query, "discovery")
	if result.(model.Matrix).Len() == 0 {
		query = `sum(kube_pod_container_resource_limits_cpu_cores) by (pod,namespace,container)*1000`
		result, err = common.MetricCollect(args, query, "discovery")
		if err != nil {
			args.WarnLogger.Println("metric=cpuLimit query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cpuLimit query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "cpuLimit")
		}

		query = `sum(kube_pod_container_resource_limits_memory_bytes) by (pod,namespace,container)/1024/1024`
		result, err = common.MetricCollect(args, query, "discovery")
		if err != nil {
			args.WarnLogger.Println("metric=memLimit query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=memLimit query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "memLimit")
		}
	} else {
		getContainerMetric(result, "pod", "container", "limits")
	}

	query = `sum(kube_pod_container_resource_requests) by (pod,namespace,container,resource)`
	result, err = common.MetricCollect(args, query, "discovery")
	if result.(model.Matrix).Len() == 0 {
		query = `sum(kube_pod_container_resource_requests_cpu_cores) by (pod,namespace,container)*1000`
		result, err = common.MetricCollect(args, query, "discovery")
		if err != nil {
			args.WarnLogger.Println("metric=cpuRequest query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=cpuRequest query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "cpuRequest")
		}

		query = `sum(kube_pod_container_resource_requests_memory_bytes) by (pod,namespace,container)/1024/1024`
		result, err = common.MetricCollect(args, query, "discovery")
		if err != nil {
			args.WarnLogger.Println("metric=memRequest query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=memRequest query=" + query + " message=" + err.Error())
		} else {
			getContainerMetric(result, "pod", "container", "memRequest")
		}
	} else {
		getContainerMetric(result, "pod", "container", "requests")
	}

	query = `container_spec_cpu_shares`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=conLabel query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=conLabel query=" + query + " message=" + err.Error())
	} else {
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
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=podInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podInfo query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "pod", "label", "Pods", query)
	}

	query = `kube_pod_labels`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=podLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=podLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "pod", "label", "Pods", query)
	}

	query = `sum(kube_pod_container_status_terminated) by (pod,namespace,container)`
	result, err = common.MetricCollect(args, query, "discovery")
	if result.(model.Matrix).Len() == 0 {
		query = `sum(kube_pod_container_status_terminated_reason) by (pod,namespace,container)`
		result, err = common.MetricCollect(args, query, "discovery")
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
	result, err = common.MetricCollect(args, query, "discovery")
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
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=namespaceLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=namespaceLabels query=" + query + " message=" + err.Error())
	} else {
		getNamespaceMetric(result, query)
	}

	query = `kube_namespace_annotations`
	result, err = common.MetricCollect(args, query, "discovery")
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
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=labels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=labels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "deployment", "label", "Deployment", query)
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_surge`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=maxSurge query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=maxSurge query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "deployment", query, "Deployment", query)
	}

	query = `kube_deployment_spec_strategy_rollingupdate_max_unavailable`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=maxUnavailable query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=maxUnavailable query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "deployment", query, "Deployment", query)
	}

	query = `kube_deployment_metadata_generation`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=metadataGeneration query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=metadataGeneration query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "deployment", query, "Deployment", query)
	}

	query = `kube_deployment_created`
	result, err = common.MetricCollect(args, query, "discovery")
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
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=replicaSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=replicaSetLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "replicaset", "label", "ReplicaSet", query)
	}

	query = `kube_replicaset_created`
	result, err = common.MetricCollect(args, query, "discovery")
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
	result, err = common.MetricCollect(args, query, "discovery")
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
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=daemonSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=daemonSetLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "daemonset", "label", "DaemonSet", query)
	}

	query = `kube_daemonset_created`
	result, err = common.MetricCollect(args, query, "discovery")
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
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=statefulSetLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=statefulSetLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "statefulset", "label", "StatefulSet", query)
	}

	query = `kube_statefulset_created`
	result, err = common.MetricCollect(args, query, "discovery")
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
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=jobInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobInfo query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", "label", "Job", query)
	}

	query = `kube_job_labels`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=jobLabel query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobLabel query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", "label", "Job", query)
	}

	query = `kube_job_spec_completions`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=jobSpecCompletions query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobSpecCompletions query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", query, "Job", query)
	}

	query = `kube_job_spec_parallelism`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=jobSpecParallelism query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobSpecParallelism query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", query, "Job", query)
	}

	query = `kube_job_status_completion_time`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=jobStatusCompletionTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobStatusCompletionTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", query, "Job", query)
	}

	query = `kube_job_status_start_time`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=jobStatusStartTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=jobStatusStartTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "job_name", "creationTime", "Job", query)
	}

	query = `kube_job_created`
	result, err = common.MetricCollect(args, query, "discovery")
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
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=cronJobLabels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobLabels query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", "label", "CronJob", query)
	}

	query = `kube_cronjob_info`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=cronJobInfo query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobInfo query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", "label", "CronJob", query)
	}

	query = `kube_cronjob_next_schedule_time`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=cronJobNextScheduleTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobNextScheduleTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", query, "CronJob", query)
	}

	query = `kube_cronjob_status_last_schedule_time`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=cronJobStatusLastScheduleTime query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobStatusLastScheduleTime query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", query, "CronJob", query)
	}

	query = `kube_cronjob_status_active`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=cronJobStatusActive query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=cronJobStatusActive query=" + query + " message=" + err.Error())
	} else {
		getMidMetric(result, "cronjob", query, "CronJob", query)
	}

	query = `kube_cronjob_created`
	result, err = common.MetricCollect(args, query, "discovery")
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
	query = `kube_hpa_labels`
	result, err = common.MetricCollect(args, query, "discovery")

	var hpaName string
	var hpaLabel model.LabelName
	if result.(model.Matrix).Len() != 0 {
		hpaName = "hpa"
		hpaLabel = "hpa"
	} else {
		hpaName = "horizontalpodautoscaler"
		hpaLabel = "horizontalpodautoscaler"
		query = `kube_` + hpaName + `_labels`
		result, err = common.MetricCollect(args, query, "discovery")
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
