//Package hpa collects data related to containers and formats into csv files to send to Densify.
package hpa

import (
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
//"github.com/prometheus/common/model"

//namespace is used to hold information related to the namespaces defined in Kubernetes
type namespace struct {
	namespace string
	//cpuLimit, cpuRequest, memLimit, memRequest int
	hpas map[string]*hpa
}

type hpa struct {
	hpa, hpaLabel                                                        string
	maxReplicas, minReplicas, ableToScale, scalingActive, scalingLimited int
	creationTime                                                         int64
}

//Map that labels and values will be stored in
var namespaces = map[string]*namespace{}

//Metrics a global func for collecting hpa level metrics in prometheus
func Metrics(clusterName, promProtocol, promAddr, promPort, interval string, intervalSize, history int, debug bool, currentTime time.Time) {
	//Setup variables used in the code.
	var historyInterval time.Duration
	historyInterval = 0
	var promaddress, query string
	var result model.Value
	var start, end time.Time

	//Start and end time + the prometheus address used for querying
	start, end = prometheus.TimeRange(interval, intervalSize, currentTime, historyInterval)
	promaddress = promProtocol + "://" + promAddr + ":" + promPort

	//Query and store kubernetes hpa information/labels
	query = "max(kube_hpa_labels) by (hpa, namespace)"
	//query = "max by (hpa, instance, pod) (kube_pod_info)"
	result = prometheus.MetricCollect(promaddress, query, start, end)

	//Prefix for indexing (less clutter on screen)
	var rsltIndex = result.(model.Matrix)
	var namespaceList []string
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			inList := false
			for j := 0; j < len(namespaceList); j++ {
				if namespaceList[j] == string(rsltIndex[i].Metric["namespace"]) {
					inList = true
				}
			}
			if !inList {
				namespaceList = append(namespaceList, string(rsltIndex[i].Metric["namespace"]))
			}
		}
		for i := 0; i < len(namespaceList); i++ {
			namespaces[namespaceList[i]] =
				&namespace{
					namespace: string(rsltIndex[i].Metric["namespace"]),
					hpas:      map[string]*hpa{}}

		}
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			namespaces[string(rsltIndex[i].Metric["namespace"])].hpas[string(rsltIndex[i].Metric["hpa"])] =
				&hpa{
					hpa: string(rsltIndex[i].Metric["hpa"])}
		}
	}

	query = `kube_hpa_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getHPAMetricString(result, "namespace", "hpa", "hpaLabel")

	query = `kube_hpa_spec_max_replicas`
	getWorkload(promaddress, "max_replicas", "Max Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `kube_hpa_spec_min_replicas`
	getWorkload(promaddress, "min_replicas", "Min Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	/*
		query = `kube_hpa_status_condition{status="AbleToScale",condition="true"}`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getHPAMetric(result, "namespace", "hpa", "ableToScale")

		query = `kube_hpa_status_condition{status="ScalingActive",condition="true"}`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getHPAMetric(result, "namespace", "hpa", "scalingActive")
	*/
	query = `kube_hpa_status_condition{status="ScalingLimited",condition="true"}`
	getWorkload(promaddress, "condition_scaling_limited", "Scaling Limited", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `kube_hpa_status_current_replicas`
	getWorkload(promaddress, "current_replicas", "Current Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `kube_hpa_status_desired_replicas`
	getWorkload(promaddress, "desired_replicas", "Desired Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	writeConfig(clusterName, promAddr)
	writeAttributes(clusterName, promAddr)
}
