// Package resourcequota collects data related to resource quotas and formats into json files to send to Densify.
package resourcequota

import (
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

const (
	rqKey = "resourcequota"
)

var resourceQuotas = make(map[string]map[string]*datamodel.ResourceQuota)

//Metrics a global func for collecting quota level metrics in prometheus
func Metrics(args *prometheus.Parameters) {
	//Setup variables used in the code.
	var query string
	var result model.Value
	var err error

	query = `kube_resourcequota_created`
	result, err = prometheus.MetricCollect(args, query)

	if err != nil {
		args.ErrorLogger.Println("metric=resourceQuotas query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=resourceQuotas query=" + query + " message=" + err.Error())
		return
	}

	mat := result.(model.Matrix)
	n := mat.Len()
	for i := 0; i < n; i++ {
		nsName := string(mat[i].Metric[prometheus.NamespaceKey])
		rqName := string(mat[i].Metric[rqKey])
		var ok bool
		if _, ok = resourceQuotas[nsName]; !ok {
			resourceQuotas[nsName] = make(map[string]*datamodel.ResourceQuota)
		}
		var rq *datamodel.ResourceQuota
		if rq, ok = resourceQuotas[nsName][rqName]; !ok {
			rq = &datamodel.ResourceQuota{CreationTime: &datamodel.Labels{}}
			resourceQuotas[nsName][rqName] = rq
		}
		_ = rq.CreationTime.AppendSampleStreamWithValue(mat[i], "", datamodel.TimeStampConverter())
	}

	if disc, err := args.ToDiscovery(prometheus.RQEntityKind); err == nil {
		discovery := &datamodel.ResourceQuotaDiscovery{Discovery: disc, Namespaces: resourceQuotas}
		prometheus.WriteDiscovery(args, discovery, prometheus.RQEntityKind)
	}

	query = `kube_resourcequota`
	prometheus.GetWorkload(query, query, args, prometheus.RQEntityKind)
}
