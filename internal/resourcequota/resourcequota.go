package resourcequota

import (
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"
	"github.com/prometheus/common/model"
)

var resourceQuotas = map[string]map[string]time.Time{}
var entityKind = "rq"

//Metrics a global func for collecting quota level metrics in prometheus
func Metrics(args *common.Parameters) {
	//Setup variables used in the code.
	var query string
	var result model.Value
	var err error

	query = `max(kube_resourcequota_created) by (namespace,resourcequota)`
	result, err = common.MetricCollect(args, query, "discovery")

	if err != nil {
		args.ErrorLogger.Println("metric=resourceQuotas query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=resourceQuotas query=" + query + " message=" + err.Error())
		return
	}
	var rsltIndex = result.(model.Matrix)
	for i := 0; i < rsltIndex.Len(); i++ {

		namespaceName := string(result.(model.Matrix)[i].Metric["namespace"])
		unixTimeInt := int64(rsltIndex[i].Values[len(rsltIndex[i].Values)-1].Value)
		if _, ok := resourceQuotas[namespaceName]; !ok {
			resourceQuotas[namespaceName] = map[string]time.Time{}
		}
		if _, ok := resourceQuotas[namespaceName][string(rsltIndex[i].Metric["resourcequota"])]; !ok {
			resourceQuotas[namespaceName][string(rsltIndex[i].Metric["resourcequota"])] = time.Unix(unixTimeInt, 0)
		}

	}

	var cluster = map[string]*datamodel.RQCluster{}
	cluster["cluster"] = &datamodel.RQCluster{Namespaces: resourceQuotas, Name: *args.ClusterName}
	common.WriteDiscovery(args, cluster, entityKind)

	query = `kube_resourcequota`
	common.GetWorkload("kube_resourcequota", query, args, entityKind)

}
