package crq

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

type crq struct {
	//Labels & general information about each node
	labelMap map[string]string

	selectorType, selectorKey, selectorValue   string
	resources, namespaces                      string
	cpuLimit, cpuRequest, memLimit, memRequest float64
	podsLimit                                  int64
	createTime                                 time.Time
}

var crqs map[string]*crq

var entityKind = "crq"

func extractCRQAttributes(result model.Value) {
	for _, val := range result.(model.Matrix) {
		crqNameLabel, ok := val.Metric["name"]
		if !ok {
			continue
		}

		crqName := string(crqNameLabel)
		if _, ok = crqs[crqName]; !ok {
			continue
		}

		for labelName, labelVal := range val.Metric {
			switch labelName {
			case "type":
				crqs[crqName].selectorType = string(labelVal)
			case "key":
				crqs[crqName].selectorKey = string(labelVal)
			case "value":
				crqs[crqName].selectorValue = string(labelVal)
			case "namespace":
				crqs[crqName].namespaces += string(labelVal) + "|"
			default: //Do nothing
			}
		}
	}
}

func getExistingQuotas(result model.Value) {
	for _, val := range result.(model.Matrix) {
		crqNameLabel, ok := val.Metric["name"]
		if !ok {
			continue
		}
		crqName := string(crqNameLabel)

		if _, ok := crqs[crqName]; !ok {
			continue
		}

		resourceLabel, ok := val.Metric["resource"]
		if !ok {
			continue
		}
		resource := string(resourceLabel)

		crqs[crqName].resources += resource + ": " + val.Values[0].Value.String() + "|"

		if len(val.Values) < 1 {
			continue
		}
		switch resource {
		case "limits.cpu":
			crqs[crqName].cpuLimit = float64(val.Values[0].Value) * 1000
		case "requests.cpu":
			crqs[crqName].cpuRequest = float64(val.Values[0].Value) * 1000
		case "limits.memory":
			crqs[crqName].memLimit = float64(val.Values[0].Value) / (1024 * 1024)
		case "requests.memory":
			crqs[crqName].memRequest = float64(val.Values[0].Value) / (1024 * 1024)
		case "pods":
			crqs[crqName].podsLimit = int64(val.Values[0].Value)
		default:
		}
	}
}

//populateLabelMap is used to parse the label based results from Prometheus related to CRQ Entities and store them in the system's data structure.
func populateLabelMap(result model.Value, nameLabel model.LabelName) {
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		crqName, ok := result.(model.Matrix)[i].Metric[nameLabel]
		if !ok {
			continue
		}
		if _, ok := crqs[string(crqName)]; !ok {
			continue
		}
		for key, value := range result.(model.Matrix)[i].Metric {
			common.AddToLabelMap(string(key), string(value), crqs[string(crqName)].labelMap)
		}
	}
}

//writeNodeGroupConfig will create the config.csv file that is will be sent to Densify by the Forwarder.
func writeConfig(args *common.Parameters) {

	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/crq/config.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=crq message=" + err.Error())
		fmt.Println("[ERROR] entity=crq message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,crq")

	for crqName := range crqs {
		fmt.Fprintf(configWrite, "%s,%s,", *args.ClusterName, crqName)
	}
	configWrite.Close()
}

func writeAttributes(args *common.Parameters) {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/crq/attributes.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=crq message=" + err.Error())
		fmt.Println("[ERROR] entity=crq message=" + err.Error())
		return
	}

	defer attributeWrite.Close()

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,crq,Quota Labels,Quota Selector Type,Quota Selector Key,Quota Selector Value,Quoted Resources,Existing CPU Limit mCores,Existing CPU Request mCores,Existing Memory Limit MB,Existing Memory Request MB,Existing Pods Limit,Namespaces Affected")

	//Loop through the CRQs and write out the attributes data for each system.
	for crqName, crq := range crqs {

		//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
		fmt.Fprintf(attributeWrite, "%s,%s,%s", *args.ClusterName, crqName, crq.createTime.Format("2006-01-02 15:04:05.000"))

		for key, value := range crq.labelMap {
			if len(key) >= 250 {
				continue
			}
			value = strings.Replace(value, ",", " ", -1)
			if len(value)+3+len(key) < 256 {
				fmt.Fprintf(attributeWrite, key+" : "+value+"|")
			} else {
				templength := 256 - 3 - len(key)
				fmt.Fprintf(attributeWrite, key+" : "+value[:templength]+"|")
			}
		}

		fmt.Fprintf(attributeWrite, ",%s,%s,%s,%s", crq.selectorType, crq.selectorKey, crq.selectorValue, crq.resources)

		if crq.cpuLimit == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%f", crq.cpuLimit)
		}

		if crq.cpuRequest == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%f", crq.cpuRequest)
		}

		if crq.memLimit == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%f", crq.memLimit)
		}

		if crq.memRequest == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%f", crq.memRequest)
		}

		if crq.podsLimit == -1 {
			fmt.Fprintf(attributeWrite, ",")
		} else {
			fmt.Fprintf(attributeWrite, ",%d", crq.podsLimit)
		}

		fmt.Fprintf(attributeWrite, ",%s\n", crq.namespaces)
	}
}

//Metrics a global func for collecting quota level metrics in prometheus
func Metrics(args *common.Parameters) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var query string
	var result model.Value
	var err error

	//Start and end time + the prometheus address used for querying
	range5Min := common.TimeRange(args, historyInterval)

	query = `max(openshift_clusterresourcequota_created) by (namespace,name)`
	result, err = common.MetricCollect(args, query, range5Min)

	if err != nil {
		args.ErrorLogger.Println("metric=clusterResourceQuotas query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=clusterResourceQuotas query=" + query + " message=" + err.Error())
		return
	}
	var rsltIndex = result.(model.Matrix)
	for i := 0; i < rsltIndex.Len(); i++ {

		fmt.Println(rsltIndex[i].Values)
		unixTimeInt := int64(rsltIndex[i].Values[len(rsltIndex[i].Values)-1].Value)
		if err != nil {
			fmt.Println("ERROR: Unable to parse unix time into int")
		}
		crqs[string(rsltIndex[i].Metric["name"])] =
			&crq{
				labelMap: map[string]string{},

				cpuLimit: -1, cpuRequest: -1, memLimit: -1, memRequest: -1, podsLimit: -1,
				createTime: time.Unix(unixTimeInt, 0),
			}
	}

	query = `max(openshift_clusterresourcequota_selector) by (name, key, type, value)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=openshift_clusterresourcequota_selector query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=openshift_clusterresourcequota_selector query=" + query + " message=" + err.Error())
	} else {
		extractCRQAttributes(result)
	}

	query = `openshift_clusterresourcequota_labels`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=openshift_clusterresourcequota_labels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=openshift_clusterresourcequota_labels query=" + query + " message=" + err.Error())
	} else {
		populateLabelMap(result, "name")
	}

	query = `max(openshift_clusterresourcequota_usage{type="hard"}) by (name, resource)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=openshift_clusterresourcequota_usage query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=openshift_clusterresourcequota_usage query=" + query + " message=" + err.Error())
	} else {
		getExistingQuotas(result)
	}

	query = `max(openshift_clusterresourcequota_namespace_usage) by (name, namespace)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=openshift_clusterresourcequota_namespace_usage query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=openshift_clusterresourcequota_namespace_usage query=" + query + " message=" + err.Error())
	} else {
		extractCRQAttributes(result)
	}

	writeAttributes(args)
	writeConfig(args)

	query = `sum(openshift_clusterresourcequota_usage{type="used", resource="limits.cpu"}) by (name) * 1000`
	common.GetWorkload("cpu_limit", "CPU Utilization in mCores", query, "name", args, entityKind)

	query = `sum(openshift_clusterresourcequota_usage{type="used", resource="requests.cpu"}) by (name) * 1000`
	common.GetWorkload("cpu_request", "Prometheus CPU Utilization in mCores", query, "name", args, entityKind)

	query = `sum(openshift_clusterresourcequota_usage{type="used", resource="limits.memory"}) by (name) / (1024 * 1024)`
	common.GetWorkload("mem_limit", "Raw Mem Utilization", query, "name", args, entityKind)

	query = `sum(openshift_clusterresourcequota_usage{type="used", resource="requests.memory"}) by (name) / (1024 * 1024)`
	common.GetWorkload("mem_request", "Prometheus Raw Mem Utilization", query, "name", args, entityKind)

	query = `sum(openshift_clusterresourcequota_usage{type="used", resource="pods"}) by (name)`
	common.GetWorkload("pods", "Auto Scaling - In Service Instances", query, "name", args, entityKind)

}
