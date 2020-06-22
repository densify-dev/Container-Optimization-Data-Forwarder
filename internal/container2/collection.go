//Package container2 collects data related to containers and formats into csv files to send to Densify.
package container2

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//getContainerMetric is used to parse the results from Prometheus related to Container Entities and store them in the systems data structure.
func getContainerMetric(result model.Value, namespace, pod, container model.LabelName, metric string) bool {
	var status bool = false
	//Validate there is data in the results.
	if result == nil {
		return status
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Validate that the data contains the namespace label with value and check it exists in our systems structure.
		namespaceValue, test := result.(model.Matrix)[i].Metric[namespace]
		if test == false {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; ok == false {
			continue
		}
		//Validate that the data contains the pod label with value and check it exists in our systems structure
		podValue, test := result.(model.Matrix)[i].Metric[pod]
		if test == false {
			continue
		}
		if _, ok := systems[string(namespaceValue)].pointers["Pod__"+string(podValue)]; ok == false {
			continue
		}
		//Validate that the data contains the container label with value and check it exists in our systems structure
		containerValue, test := result.(model.Matrix)[i].Metric[container]
		if test == false {
			continue
		}
		if _, ok := systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)]; ok == false {
			continue
		}
		//validates that the value of the entity is set and if not will default to 0
		var value int
		if len(result.(model.Matrix)[i].Values) == 0 {
			value = 0
		} else {
			value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
		}

		//Check which metric this is for and update the corresponding variable for this container in the system data structure
		switch metric {
		case "memory":
			systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].memory = value
		case "cpuLimit":
			systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].cpuLimit = value
		case "cpuRequest":
			systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].cpuRequest = value
		case "memLimit":
			systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].memLimit = value
		case "memRequest":
			systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].memRequest = value
		case "restarts":
			systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].restarts = value
		case "powerState":
			systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].powerState = value
		}
		status = true
	}
	return status
}

//getContainerMetricString is used to parse the label based results from Prometheus related to Container Entities and store them in the systems data structure.
func getContainerMetricString(result model.Value, namespace model.LabelName, pod, container model.LabelName) {
	//Validate there is data in the results.
	if result == nil {
		return
	}
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
		namespaceValue, test := result.(model.Matrix)[i].Metric[namespace]
		if test == false {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; ok == false {
			continue
		}
		//Validate that the data contains the pod label with value and check it exists in our temp structure if not it will be added
		podValue, test := result.(model.Matrix)[i].Metric[pod]
		if test == false {
			continue
		}
		if _, ok := systems[string(namespaceValue)].pointers["Pod__"+string(podValue)]; ok == false {
			continue
		}
		//Validate that the data contains the container label with value and check it exists in our temp structure if not it will be added
		containerValue, test := result.(model.Matrix)[i].Metric[container]
		if test == false {
			continue
		}
		if _, ok := systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)]; ok == false {
			continue
		}
		//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
		for key, value := range result.(model.Matrix)[i].Metric {
			addToLabelMap(string(key), string(value), systems[string(namespaceValue)].pointers["Pod__"+string(podValue)].containers[string(containerValue)].labelMap)
		}
	}
}

//getmidMetric is used to parse the results from Prometheus related to mid Entities and store them in the systems data structure.
func getMidMetric(result model.Value, namespace model.LabelName, mid model.LabelName, metric string, prefix string) {
	//Validate there is data in the results.
	if result == nil {
		return
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Validate that the data contains the namespace label with value and check it exists in our systems structure.
		namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]
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
		if _, ok := systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)]; !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)].midLevels[prefix+"__"+string(midValue)]; !ok {
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
		case "currentSize":
			systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].currentSize = int(value)
		case "creationTime":
			systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].creationTime = value
		default:
			addToLabelMap(metric, strconv.FormatInt(value, 10), systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].labelMap)
		}
	}
}

//getmidMetricString is used to parse the label based results from Prometheus related to mid Entities and store them in the systems data structure.
func getMidMetricString(result model.Value, namespace model.LabelName, mid model.LabelName, prefix string) {
	//temp structure used to store data while working with it. As we are combining the labels into a formatted string for loading.
	//Validate there is data in the results.
	if result == nil {
		return
	}
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
		namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; !ok {
			continue
		}
		//Validate that the data contains the mid label with value and check it exists in our temp structure if not it will be added
		midValue, ok := result.(model.Matrix)[i].Metric[mid]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)]; !ok {
			continue
		}
		//loop through all the labels for an entity and store them in a map. For controller based entities where there will be multiple copies of containers they will have there values concatinated together.
		for key, value := range result.(model.Matrix)[i].Metric {
			addToLabelMap(string(key), string(value), systems[string(namespaceValue)].pointers[prefix+"__"+string(midValue)].labelMap)
		}
	}
}

//getHPAMetricString is used to parse the label based results from Prometheus related to mid Entities and store them in the systems data structure.
func getHPAMetricString(result model.Value, namespace model.LabelName, hpa model.LabelName, args *common.Parameters) {
	hpas := map[string]map[string]string{}
	//Validate there is data in the results.
	if result == nil {
		return
	}
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
		namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; !ok {
			continue
		}
		//Validate that the data contains the mid label with value and check it exists in our temp structure if not it will be added
		hpaValue, ok := result.(model.Matrix)[i].Metric[hpa]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)].pointers["Deployment__"+string(hpaValue)]; ok {
			for key, value := range result.(model.Matrix)[i].Metric {
				addToLabelMap(string(key), string(value), systems[string(namespaceValue)].pointers["Deployment__"+string(hpaValue)].labelMap)
			}
		} else if _, ok := systems[string(namespaceValue)].pointers["ReplicaSet__"+string(hpaValue)]; ok {
			for key, value := range result.(model.Matrix)[i].Metric {
				addToLabelMap(string(key), string(value), systems[string(namespaceValue)].pointers["ReplicaSet__"+string(hpaValue)].labelMap)
			}
		} else if _, ok := systems[string(namespaceValue)].pointers["ReplicationController__"+string(hpaValue)]; ok {
			for key, value := range result.(model.Matrix)[i].Metric {
				addToLabelMap(string(key), string(value), systems[string(namespaceValue)].pointers["ReplicationController__"+string(hpaValue)].labelMap)
			}
		} else {
			hpas[string(hpaValue)] = map[string]string{}
			for key, value := range result.(model.Matrix)[i].Metric {
				addToLabelMap(string(key), string(value), hpas[string(hpaValue)])
			}
		}
	}
	if len(hpas) > 0 {
		writeHPAAttributes(args, hpas)
		writeHPAConfig(args, hpas)
	}
}

//getNamespaceMetric is used to parse the results from Prometheus related to Namespace Entities and store them in the systems data structure.
func getNamespacelimits(result model.Value, namespace model.LabelName) {
	//Validate there is data in the results.
	if result == nil {
		return
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Validate that the data contains the namespace label with value and check it exists in our systems structure.
		namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; !ok {
			continue
		}
		//validates that the value of the entity is set and if not will default to 0
		var value int = 0
		if len(result.(model.Matrix)[i].Values) != 0 {
			value = int(result.(model.Matrix)[i].Values[len(result.(model.Matrix)[i].Values)-1].Value)
		}

		//Check which metric this is for and update the corresponding variable for this container in the system data structure
		//For systems limits they are defined based on 2 of the labels as they combine the Limits and Request for CPU and Memory all into 1 call.
		constraint := result.(model.Matrix)[i].Metric["constraint"]
		resource := result.(model.Matrix)[i].Metric["resource"]
		switch {
		case constraint == "defaultRequest" && resource == "cpu":
			systems[string(namespaceValue)].cpuRequest = value
		case constraint == "defaultRequest" && resource == "memory":
			systems[string(namespaceValue)].memRequest = value
		case constraint == "default" && resource == "cpu":
			systems[string(namespaceValue)].cpuLimit = value
		case constraint == "default" && resource == "memory":
			systems[string(namespaceValue)].memLimit = value
		}
	}
}

//getNamespaceMetricString is used to parse the label based results from Prometheus related to Namespace Entities and store them in the systems data structure.
func getNamespaceMetricString(result model.Value, namespace model.LabelName) {
	//Validate there is data in the results.
	if result == nil {
		return
	}
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		//Validate that the data contains the namespace label with value and check it exists in our temp structure if not it will be added.
		namespaceValue, ok := result.(model.Matrix)[i].Metric[namespace]
		if !ok {
			continue
		}
		if _, ok := systems[string(namespaceValue)]; !ok {
			continue
		}
		//loop through all the labels for an entity and store them in a map.
		for key, value := range result.(model.Matrix)[i].Metric {
			addToLabelMap(string(key), string(value), systems[string(namespaceValue)].labelMap)
		}
	}
}

func getWorkload(fileName, metricName, query, aggregator string, args *common.Parameters) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	var query2 string

	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/container/" + aggregator + `_` + fileName + ".csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " metric=" + metricName + " query=" + query + " message=" + err.Error())
		fmt.Println("entity=" + entityKind + " metric=" + metricName + " query=" + query + " message=" + err.Error())
		return
	}
	fmt.Fprintf(workloadWrite, "cluster,namespace,entity_name,entity_type,container,Datetime,%s\n", metricName)

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slpwer prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < *args.History; historyInterval++ {
		range5Min := prometheus.TimeRange(args, historyInterval)

		//query containers under a pod with no owner
		query2 = aggregator + `(` + query + ` * on (pod, namespace) group_left max(kube_pod_owner{owner_name="<none>"}) by (namespace, pod, container` + args.LabelSuffix + `)) by (pod,namespace,container` + args.LabelSuffix + `)`
		result = prometheus.MetricCollect(args, query2, range5Min, "pod_"+metricName, false)
		writeWorkload(workloadWrite, result, "namespace", "pod", model.LabelName("container"+args.LabelSuffix), args, "Pod")

		//query containers under a controller with no owner
		query2 = aggregator + `(` + query + ` * on (pod, namespace) group_left (owner_name,owner_kind) max(kube_pod_owner) by (namespace, pod, owner_name, owner_kind)) by (owner_kind,owner_name,namespace,container` + args.LabelSuffix + `)`
		result = prometheus.MetricCollect(args, query2, range5Min, "controller_"+metricName, false)
		writeWorkload(workloadWrite, result, "namespace", "owner_name", model.LabelName("container"+args.LabelSuffix), args, "")

		//query containers under a deployment
		query2 = aggregator + `(` + query + ` * on (pod, namespace) group_left (replicaset) max(label_replace(kube_pod_owner{owner_kind="ReplicaSet"}, "replicaset", "$1", "owner_name", "(.*)")) by (namespace, pod, replicaset) * on (replicaset, namespace) group_left (owner_name) max(kube_replicaset_owner{owner_kind="Deployment"}) by (namespace, replicaset, owner_name)) by (owner_name,namespace,container` + args.LabelSuffix + `)`
		result = prometheus.MetricCollect(args, query2, range5Min, "deployment_"+metricName, false)
		writeWorkload(workloadWrite, result, "namespace", "owner_name", model.LabelName("container"+args.LabelSuffix), args, "Deployment")

		//query containers under a cron job
		query2 = aggregator + `(` + query + ` * on (pod, namespace) group_left (job) max(label_replace(kube_pod_owner{owner_kind="Job"}, "job", "$1", "owner_name", "(.*)")) by (namespace, pod, job) * on (job, namespace) group_left (owner_name) max(label_replace(kube_job_owner{owner_kind="CronJob"}, "job", "$1", "job_name", "(.*)")) by (namespace, job, owner_name)) by (owner_name,namespace,container` + args.LabelSuffix + `)`
		result = prometheus.MetricCollect(args, query2, range5Min, "cronJob_"+metricName, false)
		writeWorkload(workloadWrite, result, "namespace", "owner_name", model.LabelName("container"+args.LabelSuffix), args, "CronJob")

	}
	//Close the workload files.
	workloadWrite.Close()
}

func getDeploymentWorkload(fileName, metricName, query string, args *common.Parameters) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value

	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/container/deployment_" + fileName + ".csv")
	if err != nil {
		args.ErrorLogger.Println("metric=" + metricName + " query=" + query + " message=File not found")
		fmt.Println("metric=" + metricName + " query=" + query + " message=File not found")
		return
	}
	fmt.Fprintf(workloadWrite, "cluster,namespace,entity_name,entity_type,container,Datetime,%s\n", metricName)

	tempMap := map[int]map[string]map[string][]model.SamplePair{}

	for historyInterval = 0; int(historyInterval) < *args.History; historyInterval++ {
		tempMap[int(historyInterval)] = map[string]map[string][]model.SamplePair{}
		range5Min := prometheus.TimeRange(args, historyInterval)

		result = prometheus.MetricCollect(args, query, range5Min, metricName, false)
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
				if _, ok := tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])]; !ok {
					tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])] = map[string][]model.SamplePair{}
				}
				if _, ok := tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["deployment"])]; !ok {
					tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["deployment"])] = []model.SamplePair{}
				}
				tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["deployment"])] = append(tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["deployment"])], result.(model.Matrix)[i].Values[j])
			}
		}
	}

	for n := range systems {
		for m, midVal := range systems[n].midLevels {
			if midVal.kind != "Deployment" {
				continue
			}
			for c := range systems[n].midLevels[m].containers {
				for historyInterval = 0; int(historyInterval) < *args.History; historyInterval++ {
					for _, val := range tempMap[int(historyInterval)][n][midVal.name] {
						fmt.Fprintf(workloadWrite, "%s,%s,%s,%s,%s,%s,%f\n", *args.ClusterName, n, midVal.name, midVal.kind, c, time.Unix(0, int64(val.Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), val.Value)
					}
				}
			}
		}
	}
	workloadWrite.Close()
}

func getHPAWorkload(fileName, metricName, query string, args *common.Parameters) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value

	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/container/hpa_" + fileName + ".csv")
	if err != nil {
		args.ErrorLogger.Println("metric=" + metricName + " query=" + query + " message=File not found")
		fmt.Println("metric=" + metricName + " query=" + query + " message=File not found")
		return
	}
	workloadWriteExtra, err := os.Create("./data/hpa/hpa_extra_" + fileName + ".csv")
	if err != nil {
		args.ErrorLogger.Println("metric=" + metricName + " query=" + query + " message=File not found")
		fmt.Println("metric=" + metricName + " query=" + query + " message=File not found")
		return
	}
	fmt.Fprintf(workloadWrite, "cluster,namespace,entity_name,entity_type,container,HPA Name,Datetime,%s\n", metricName)
	fmt.Fprintf(workloadWriteExtra, "cluster,namespace,entity_name,entity_type,container,HPA Name,Datetime,%s\n", metricName)

	tempMap := map[int]map[string]map[string][]model.SamplePair{}

	for historyInterval = 0; int(historyInterval) < *args.History; historyInterval++ {
		tempMap[int(historyInterval)] = map[string]map[string][]model.SamplePair{}
		range5Min := prometheus.TimeRange(args, historyInterval)

		result = prometheus.MetricCollect(args, query, range5Min, metricName, false)
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
				if _, ok := tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])]; !ok {
					tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])] = map[string][]model.SamplePair{}
				}
				if _, ok := tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["hpa"])]; !ok {
					tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["hpa"])] = []model.SamplePair{}
				}
				tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["hpa"])] = append(tempMap[int(historyInterval)][string(result.(model.Matrix)[i].Metric["namespace"])][string(result.(model.Matrix)[i].Metric["hpa"])], result.(model.Matrix)[i].Values[j])
			}
		}
	}

	for historyInterval = 0; int(historyInterval) < *args.History; historyInterval++ {
		for n := range systems {
			for m, midVal := range systems[n].pointers {
				switch midVal.kind {
				case "Deployment":
					for c := range systems[n].pointers[m].containers {
						for _, val := range tempMap[int(historyInterval)][n][midVal.name] {
							fmt.Fprintf(workloadWrite, "%s,%s,%s,%s,%s,%s,%s,%f\n", *args.ClusterName, n, midVal.name, midVal.kind, c, midVal.name, time.Unix(0, int64(val.Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), val.Value)
						}
					}
				case "ReplicaSet":
					for c := range systems[n].pointers[m].containers {
						for _, val := range tempMap[int(historyInterval)][n][midVal.name] {
							fmt.Fprintf(workloadWrite, "%s,%s,%s,%s,%s,%s,%s,%f\n", *args.ClusterName, n, midVal.name, midVal.kind, c, midVal.name, time.Unix(0, int64(val.Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), val.Value)
						}
					}
				case "ReplicationController":
					for c := range systems[n].pointers[m].containers {
						for _, val := range tempMap[int(historyInterval)][n][midVal.name] {
							fmt.Fprintf(workloadWrite, "%s,%s,%s,%s,%s,%s,%s,%f\n", *args.ClusterName, n, midVal.name, midVal.kind, c, midVal.name, time.Unix(0, int64(val.Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), val.Value)
						}
					}
				}
				delete(tempMap[int(historyInterval)][n], midVal.name)
			}
		}
	}
	workloadWrite.Close()
	for historyInterval = 0; int(historyInterval) < *args.History; historyInterval++ {
		for i := range tempMap {
			for n := range tempMap[i] {
				for m := range tempMap[i][n] {
					for _, val := range tempMap[int(historyInterval)][n][m] {
						fmt.Fprintf(workloadWriteExtra, "%s,%s,,,,%s,%s,%f\n", *args.ClusterName, n, m, time.Unix(0, int64(val.Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), val.Value)
					}
				}
			}
		}
	}
	workloadWriteExtra.Close()
}

func addToLabelMap(key string, value string, labelPath map[string]string) {
	if _, ok := labelPath[key]; !ok {
		value = strings.Replace(value, "\n", "", -1)
		if len(value) > 255 {
			labelPath[key] = value[:255]
		} else {
			labelPath[key] = value
		}
		return
	}

	if strings.Contains(value, ";") {
		currValue := ""
		for _, l := range value {
			currValue = currValue + string(l)
			if l == ';' {
				addToLabelMap(key, currValue[:len(currValue)-1], labelPath)
				currValue = ""
			}
		}
		addToLabelMap(key, currValue, labelPath)
		return
	}

	currValue := ""
	notPresent := true
	for _, l := range labelPath[key] {
		currValue = currValue + string(l)
		if l == ';' {
			if currValue[:len(currValue)-1] == value {
				notPresent = false
				break
			}
			currValue = ""
		}
	}
	if currValue != value && notPresent {
		if len(value) > 255 {
			labelPath[key] = labelPath[key] + ";" + value[:255]
		} else {
			labelPath[key] = labelPath[key] + ";" + value
		}
	}
}
