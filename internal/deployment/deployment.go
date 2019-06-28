//Package deployment collects data related to containers and formats into csv files to send to Densify.
package deployment

import (
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/prometheus"
	"github.com/prometheus/common/model"
)

//namespace is used to hold information related to the namespaces defined in Kubernetes
type namespace struct {
	namespace string
	//cpuLimit, cpuRequest, memLimit, memRequest int
	deployments map[string]*deployment
}

type deployment struct {
	deployment, deploymentLabel                  string
	metadataGeneration, maxSurge, maxUnavailable int
	creationTime                                 int64
}

//Map that labels and values will be stored in
var namespaces = map[string]*namespace{}

//Metrics a global func for collecting deployment level metrics in prometheus
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

	//Query and store kubernetes deployment information/labels
	query = "max(kube_deployment_labels) by (deployment, namespace)"
	//query = "max by (deployment, instance, pod) (kube_pod_info)"
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
					namespace:   string(rsltIndex[i].Metric["namespace"]),
					deployments: map[string]*deployment{}}

		}
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			namespaces[string(rsltIndex[i].Metric["namespace"])].deployments[string(rsltIndex[i].Metric["deployment"])] =
				&deployment{
					deployment: string(rsltIndex[i].Metric["deployment"])}
		}
	}

	query = `kube_deployment_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getDeploymentMetricString(result, "namespace", "deployment", "deploymentLabel")

	query = `kube_deployment_spec_strategy_rollingupdate_max_surge`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getDeploymentMetric(result, "namespace", "deployment", "maxSurge")

	query = `kube_deployment_spec_strategy_rollingupdate_max_unavailable`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getDeploymentMetric(result, "namespace", "deployment", "maxUnavailable")

	query = `kube_deployment_metadata_generation`
	getDeploymentMetric(result, "namespace", "deployment", "metadataGeneration")

	query = `kube_deployment_status_replicas_available`
	getWorkload(promaddress, "status_replicas_available", "Status Replicas Available", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `kube_deployment_status_replicas`
	getWorkload(promaddress, "status_replicas", "Status Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	query = `kube_deployment_spec_replicas`
	getWorkload(promaddress, "spec_replicas", "Spec Replicas", query, clusterName, promAddr, interval, intervalSize, history, currentTime)

	writeConfig(clusterName, promAddr)
	writeAttributes(clusterName, promAddr)
}
