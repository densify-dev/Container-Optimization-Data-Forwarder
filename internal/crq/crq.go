package crq

import (
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

var crqs = make(map[string]*datamodel.ClusterResourceQuota)

const (
	crqKey   = "name"
	typeKey  = "type"
	keyKey   = "key"
	valueKey = "value"
)

var typeFilter = []string{typeKey}
var keyFilter = []string{keyKey}
var valueFilter = []string{valueKey}

//Metrics a global func for collecting quota level metrics in prometheus
func Metrics(args *prometheus.Parameters) {
	//Setup variables used in the code.
	var query string
	var result model.Value
	var err error

	query = `openshift_clusterresourcequota_created`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.ErrorLogger.Println("metric=clusterResourceQuotas query=" + query + " message=" + err.Error())
		fmt.Println("[ERROR] metric=clusterResourceQuotas query=" + query + " message=" + err.Error())
		return
	}

	mat := result.(model.Matrix)
	n := mat.Len()
	for i := 0; i < n; i++ {
		crqName := string(mat[i].Metric[crqKey])
		var ok bool
		var crq *datamodel.ClusterResourceQuota
		if crq, ok = crqs[crqName]; !ok {
			crq = &datamodel.ClusterResourceQuota{
				LabelMap:      make(datamodel.LabelMap),
				SelectorType:  &datamodel.Labels{},
				SelectorKey:   &datamodel.Labels{},
				SelectorValue: &datamodel.Labels{},
				CreationTime:  &datamodel.Labels{},
			}
			crqs[crqName] = crq
		}
		_ = crq.CreationTime.AppendSampleStreamWithValue(mat[i], "", datamodel.TimeStampConverter())
	}

	query = `openshift_clusterresourcequota_labels`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=openshift_clusterresourcequota_labels query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=openshift_clusterresourcequota_labels query=" + query + " message=" + err.Error())
	} else {
		mat, ok := result.(model.Matrix)
		if ok {
			n := mat.Len()
			for i := 0; i < n; i++ {
				crq, ok := getCRQ(mat[i])
				if !ok {
					continue
				}
				labels := datamodel.EnsureLabels(crq.LabelMap, query)
				_ = labels.AppendSampleStream(mat[i])
			}
		}
	}

	query = `openshift_clusterresourcequota_selector`
	result, err = prometheus.MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=openshift_clusterresourcequota_selector query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=openshift_clusterresourcequota_selector query=" + query + " message=" + err.Error())
	} else {
		mat := result.(model.Matrix)
		n := mat.Len()
		for i := 0; i < n; i++ {
			crq, ok := getCRQ(mat[i])
			if !ok {
				continue
			}
			_ = crq.SelectorType.AppendSampleStreamWithFilter(mat[i], typeFilter)
			_ = crq.SelectorKey.AppendSampleStreamWithFilter(mat[i], keyFilter)
			_ = crq.SelectorValue.AppendSampleStreamWithFilter(mat[i], valueFilter)
		}
	}
	if disc, err := args.ToDiscovery(prometheus.CRQEntityKind); err == nil {
		discovery := &datamodel.ClusterResourceQuotaDiscovery{Discovery: disc, CRQs: crqs}
		prometheus.WriteDiscovery(args, discovery, prometheus.CRQEntityKind)
	}

	query = `openshift_clusterresourcequota_usage`
	prometheus.GetWorkload(query, query, args, prometheus.CRQEntityKind)

	query = `openshift_clusterresourcequota_namespace_usage`
	prometheus.GetWorkload(query, query, args, prometheus.CRQEntityKind)
}

func getCRQ(ss *model.SampleStream) (*datamodel.ClusterResourceQuota, bool) {
	var crq *datamodel.ClusterResourceQuota
	var crqName string
	var ok bool
	if crqName, ok = getCRQName(ss); ok {
		crq, ok = crqs[crqName]
	}
	return crq, ok
}

func getCRQName(ss *model.SampleStream) (string, bool) {
	var crqName string
	name, ok := ss.Metric[crqKey]
	if ok {
		crqName = string(name)
	}
	return crqName, ok
}
