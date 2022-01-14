package crq

import (
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

var crqs = map[string]*datamodel.CRQ{}

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
				crqs[crqName].SelectorType = string(labelVal)
			case "key":
				crqs[crqName].SelectorKey = string(labelVal)
			case "value":
				crqs[crqName].SelectorValue = string(labelVal)
			case "namespace":
				crqs[crqName].Namespaces += string(labelVal) + "|"
			}
		}
	}
}

//populateLabelMap is used to parse the label based results from Prometheus related to CRQ Entities and store them in the system's data structure.
func populateLabelMap(result model.Value, nameLabel model.LabelName, label string) {
	//Loop through the different entities in the results.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		crqName, ok := result.(model.Matrix)[i].Metric[nameLabel]
		if !ok {
			continue
		}
		if _, ok := crqs[string(crqName)]; !ok {
			continue
		}
		if _, ok := crqs[string(crqName)].LabelMap[label]; !ok {
			crqs[string(crqName)].LabelMap[label] = map[string]string{}
		}
		for key, value := range result.(model.Matrix)[i].Metric {
			common.AddToLabelMap(string(key), string(value), crqs[string(crqName)].LabelMap[label])
		}
	}
}

//Metrics a global func for collecting quota level metrics in prometheus
func Metrics(args *common.Parameters) {
	//Setup variables used in the code.
	var query string
	var result model.Value
	var err error

	query = `max(openshift_clusterresourcequota_created) by (namespace,name)`
	result, err = common.MetricCollect(args, query, "discovery")

	if err != nil {
		args.ErrorLogger.Println("metric=clusterResourceQuotas query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=clusterResourceQuotas query=" + query + " message=" + err.Error())
		return
	}
	var rsltIndex = result.(model.Matrix)
	for i := 0; i < rsltIndex.Len(); i++ {

		unixTimeInt := int64(rsltIndex[i].Values[len(rsltIndex[i].Values)-1].Value)
		crqs[string(rsltIndex[i].Metric["name"])] =
			&datamodel.CRQ{
				LabelMap:     map[string]map[string]string{},
				SelectorType: "", SelectorKey: "", SelectorValue: "", Namespaces: "",
				CreateTime: time.Unix(unixTimeInt, 0),
			}
	}

	query = `max(openshift_clusterresourcequota_selector) by (name, key, type, value)`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=openshift_clusterresourcequota_selector query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=openshift_clusterresourcequota_selector query=" + query + " message=" + err.Error())
	} else {
		extractCRQAttributes(result)
	}

	query = `openshift_clusterresourcequota_labels`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=openshift_clusterresourcequota_labels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=openshift_clusterresourcequota_labels query=" + query + " message=" + err.Error())
	} else {
		populateLabelMap(result, "name", query)
	}

	query = `max(openshift_clusterresourcequota_namespace_usage) by (name, namespace)`
	result, err = common.MetricCollect(args, query, "discovery")
	if err != nil {
		args.WarnLogger.Println("metric=openshift_clusterresourcequota_namespace_usage query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=openshift_clusterresourcequota_namespace_usage query=" + query + " message=" + err.Error())
	} else {
		extractCRQAttributes(result)
	}

	var cluster = map[string]*datamodel.CRQCluster{}
	cluster["cluster"] = &datamodel.CRQCluster{CRQs: crqs, Name: *args.ClusterName}
	common.WriteDiscovery(args, cluster, entityKind)

	query = `openshift_clusterresourcequota_usage`
	common.GetWorkload("openshift_clusterresourcequota_usage", query, args, entityKind)

}
