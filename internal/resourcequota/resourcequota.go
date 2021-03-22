package resourcequota

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

type resourceQuota struct {
	resources                                                                                                                             string
	cpuLimit, cpuRequest, memLimit, memRequest, usageCpuLimit, usageCpuRequest, usageMemLimit, usageMemRequest, usagePodsLimit, podsLimit int
	createTime                                                                                                                            time.Time
}
type namespace struct {
	rqs map[string]*resourceQuota
}

var resourceQuotas = map[string]*namespace{}

var entityKind = "rq"

func getExistingQuotas(result model.Value) {
	for _, val := range result.(model.Matrix) {
		namespaceName, ok := val.Metric["namespace"]
		if !ok {
			continue
		}
		if _, ok := resourceQuotas[string(namespaceName)]; !ok {
			continue
		}

		rqName, ok := val.Metric["resourcequota"]
		if !ok {
			continue
		}

		if _, ok := resourceQuotas[string(namespaceName)].rqs[string(rqName)]; !ok {
			continue
		}

		resourceLabel, ok := val.Metric["resource"]
		if !ok {
			continue
		}
		resource := string(resourceLabel)

		var value float64
		if len(val.Values) != 0 {
			value = float64(val.Values[len(val.Values)-1].Value)
		}

		if typeHard := val.Metric["type"]; typeHard == "hard" {

			resourceQuotas[string(namespaceName)].rqs[string(rqName)].resources += resource + ": " + strconv.FormatFloat(value, 'f', 2, 64) + "|"

			switch resource {
			case "limits.cpu":
				resourceQuotas[string(namespaceName)].rqs[string(rqName)].cpuLimit = int(value * 1000)
			case "requests.cpu", "cpu":
				resourceQuotas[string(namespaceName)].rqs[string(rqName)].cpuRequest = int(value * 1000)
			case "limits.memory":
				resourceQuotas[string(namespaceName)].rqs[string(rqName)].memLimit = int(value / (1024 * 1024))
			case "requests.memory", "memory":
				resourceQuotas[string(namespaceName)].rqs[string(rqName)].memRequest = int(value / (1024 * 1024))
			case "count/pods", "pods":
				resourceQuotas[string(namespaceName)].rqs[string(rqName)].podsLimit = int(value)
			default:
			}
		} else if typeUsed := val.Metric["type"]; typeUsed == "used" {
			switch resource {
			case "limits.cpu":
				resourceQuotas[string(namespaceName)].rqs[string(rqName)].usageCpuLimit = int(value * 1000)
			case "requests.cpu", "cpu":
				resourceQuotas[string(namespaceName)].rqs[string(rqName)].usageCpuRequest = int(value * 1000)
			case "limits.memory":
				resourceQuotas[string(namespaceName)].rqs[string(rqName)].usageMemLimit = int(value / (1024 * 1024))
			case "requests.memory", "memory":
				resourceQuotas[string(namespaceName)].rqs[string(rqName)].usageMemRequest = int(value / (1024 * 1024))
			case "count/pods", "pods":
				resourceQuotas[string(namespaceName)].rqs[string(rqName)].usagePodsLimit = int(value)
			default:
			}
		}
	}
}

//writeNodeGroupConfig will create the config.csv file that is will be sent to Densify by the Forwarder.
func writeConfig(args *common.Parameters) {

	//Create the config file and open it for writing.
	configWrite, err := os.Create("./data/rq/config.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	//Write out the header.
	fmt.Fprintln(configWrite, "cluster,namespace,rq")
	for kn := range resourceQuotas {
		for krq := range resourceQuotas[kn].rqs {
			fmt.Fprintf(configWrite, "%s,%s,%s\n", *args.ClusterName, kn, krq)
		}
	}
	configWrite.Close()
}

func writeAttributes(args *common.Parameters) {
	//Create the attributes file and open it for writing
	attributeWrite, err := os.Create("./data/rq/attributes.csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		return
	}

	defer attributeWrite.Close()

	//Write out the header.
	fmt.Fprintln(attributeWrite, "cluster,namespace,rq,Virtual Technology,Virtual Domain,Virtual Datacenter,Create Time,Resource Metadata,Existing CPU Limit,Existing CPU Request,Existing Memory Limit,Existing Memory Request,Current Size,Namespace CPU Limit,Namespace CPU Request,Namespace Memory Limit,Namespace Memory Request,Namespace Pods Limit")

	//Loop through the resource quotas and write out the attributes data for each system.
	for kn := range resourceQuotas {
		for krq, vrq := range resourceQuotas[kn].rqs {

			//Write out the different fields. For fiels that are numeric we don't want to write -1 if it wasn't set so we write a blank if that is the value otherwise we write the number out.
			fmt.Fprintf(attributeWrite, "%s,%s,%s,ResourceQuota,%s,%s,%s", *args.ClusterName, kn, krq, *args.ClusterName, kn, vrq.createTime.Format("2006-01-02 15:04:05.000"))

			fmt.Fprintf(attributeWrite, ",%s", vrq.resources)

			if vrq.cpuLimit == -1 {
				fmt.Fprintf(attributeWrite, ",")
			} else {
				fmt.Fprintf(attributeWrite, ",%d", vrq.usageCpuLimit)
			}

			if vrq.cpuRequest == -1 {
				fmt.Fprintf(attributeWrite, ",")
			} else {
				fmt.Fprintf(attributeWrite, ",%d", vrq.usageCpuRequest)
			}

			if vrq.memLimit == -1 {
				fmt.Fprintf(attributeWrite, ",")
			} else {
				fmt.Fprintf(attributeWrite, ",%d", vrq.usageMemLimit)
			}

			if vrq.memRequest == -1 {
				fmt.Fprintf(attributeWrite, ",")
			} else {
				fmt.Fprintf(attributeWrite, ",%d", vrq.usageMemRequest)
			}

			if vrq.podsLimit == -1 {
				fmt.Fprintf(attributeWrite, ",")
			} else {
				fmt.Fprintf(attributeWrite, ",%d", vrq.usagePodsLimit)
			}

			if vrq.cpuLimit == -1 {
				fmt.Fprintf(attributeWrite, ",")
			} else {
				fmt.Fprintf(attributeWrite, ",%d", vrq.cpuLimit)
			}

			if vrq.cpuRequest == -1 {
				fmt.Fprintf(attributeWrite, ",")
			} else {
				fmt.Fprintf(attributeWrite, ",%d", vrq.cpuRequest)
			}

			if vrq.memLimit == -1 {
				fmt.Fprintf(attributeWrite, ",")
			} else {
				fmt.Fprintf(attributeWrite, ",%d", vrq.memLimit)
			}

			if vrq.memRequest == -1 {
				fmt.Fprintf(attributeWrite, ",")
			} else {
				fmt.Fprintf(attributeWrite, ",%d", vrq.memRequest)
			}

			if vrq.podsLimit == -1 {
				fmt.Fprintf(attributeWrite, ",\n")
			} else {
				fmt.Fprintf(attributeWrite, ",%d\n", vrq.podsLimit)
			}
		}
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

	query = `max(kube_resourcequota_created) by (namespace,resourcequota)`
	result, err = common.MetricCollect(args, query, range5Min)

	if err != nil {
		args.ErrorLogger.Println("metric=resourceQuotas query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=resourceQuotas query=" + query + " message=" + err.Error())
		return
	}
	var rsltIndex = result.(model.Matrix)
	for i := 0; i < rsltIndex.Len(); i++ {

		namespaceName := string(result.(model.Matrix)[i].Metric["namespace"])
		if _, ok := resourceQuotas[namespaceName]; !ok {
			resourceQuotas[namespaceName] = &namespace{rqs: map[string]*resourceQuota{}}
		}

		unixTimeInt := int64(rsltIndex[i].Values[len(rsltIndex[i].Values)-1].Value)
		if err != nil {
			fmt.Println("ERROR: Unable to parse unix time into int")
		}
		resourceQuotas[namespaceName].rqs[string(rsltIndex[i].Metric["resourcequota"])] =
			&resourceQuota{
				resources: "",
				cpuLimit:  -1, cpuRequest: -1, memLimit: -1, memRequest: -1, podsLimit: -1,
				createTime: time.Unix(unixTimeInt, 0),
			}
	}

	query = `max(kube_resourcequota) by (resourcequota, resource, namespace, type)`
	result, err = common.MetricCollect(args, query, range5Min)
	if err != nil {
		args.WarnLogger.Println("metric=resourceQuotaLimits query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=resourceQuotaLimits query=" + query + " message=" + err.Error())
	} else {
		getExistingQuotas(result)
	}

	writeAttributes(args)
	writeConfig(args)

	var metricField []model.LabelName
	metricField = append(metricField, "namespace")
	metricField = append(metricField, "resourcequota")

	query = `sum(kube_resourcequota{type="used", resource="limits.cpu"}) by (resourcequota,namespace) * 1000`
	common.GetWorkload("cpu_limit", "CPU Utilization in mCores", query, metricField, args, entityKind)

	query = `sum(kube_resourcequota{type="used", resource="requests.cpu"}) by (resourcequota,namespace) * 1000`
	common.GetWorkload("cpu_request", "Prometheus CPU Utilization in mCores", query, metricField, args, entityKind)

	query = `sum(kube_resourcequota{type="used", resource="limits.memory"}) by (resourcequota,namespace) / (1024 * 1024)`
	common.GetWorkload("mem_limit", "Raw Mem Utilization", query, metricField, args, entityKind)

	query = `sum(kube_resourcequota{type="used", resource="requests.memory"}) by (resourcequota,namespace) / (1024 * 1024)`
	common.GetWorkload("mem_request", "Prometheus Raw Mem Utilization", query, metricField, args, entityKind)

	query = `sum(kube_resourcequota{type="used", resource="count/pods"}) by (resourcequota,namespace)`
	common.GetWorkload("pods", "Auto Scaling - In Service Instances", query, metricField, args, entityKind)

}
