//Package cronjob collects data related to containers and formats into csv files to send to Densify.
package cronjob

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
	cronjobs map[string]*cronjob
}

type cronjob struct {
	cronjob, cronjobInfo, cronjobLabel               string
	statusActive, nextScheduleTime, lastScheduleTime int
	creationTime                                     int64
	jobs                                             map[string]*job
}

type job struct {
	job, jobInfo, jobLabel                                                                                                         string
	specCompletions, specParallelism, statusActive, statusFailed, statusSucceeded, statusStartTime, statusCompletionTime, complete int
}

//Map that labels and values will be stored in
var namespaces = map[string]*namespace{}

//Metrics a global func for collecting cronjob level metrics in prometheus
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

	//Query and store kubernetes cronjob information/labels
	query = "kube_job_owner"
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
					cronjobs:  map[string]*cronjob{}}
		}
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			namespaces[string(rsltIndex[i].Metric["namespace"])].cronjobs[string(rsltIndex[i].Metric["owner_name"])] =
				&cronjob{
					cronjob: string(rsltIndex[i].Metric["job_name"]),
					jobs:    map[string]*job{}}
		}
	}
	/*
		//Query and store kubernetes job information/labels
		query = `kube_job_owner`
		result = prometheus.MetricCollect(promaddress, query, start, end)*/

	//Prefix for indexing (less clutter on screen)
	rsltIndex = result.(model.Matrix)
	if result != nil {
		for i := 0; i < result.(model.Matrix).Len(); i++ {
			namespaces[string(rsltIndex[i].Metric["namespace"])].cronjobs[string(rsltIndex[i].Metric["owner_name"])].jobs[string(rsltIndex[i].Metric["job_name"])] =
				&job{
					job: string(rsltIndex[i].Metric["job_name"])}
		}
	} /*
		for i := range namespaces {
			for j := range namespaces[i].cronjobs {
				for k := range namespaces[i].cronjobs[j].jobs {
					fmt.Println(i + " --- " + j + " --- " + k)
				}
			}
		}*/

	query = `kube_cronjob_labels`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getCronJobMetricString(result, "namespace", "cronjob", "cronjobLabel")

	query = `kube_cronjob_info`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getCronJobMetricString(result, "namespace", "cronjob", "cronjobInfo")

	query = `kube_cronjob_next_schedule_time`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getCronJobMetric(result, "namespace", "cronjob", "nextScheduleTime")

	query = `kube_cronjob_status_last_schedule_time`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getCronJobMetric(result, "namespace", "cronjob", "lastScheduleTime")

	query = `kube_cronjob_status_active`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getCronJobMetric(result, "namespace", "cronjob", "statusActive")

	query = `kube_job_info * on (job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getJobMetricString(result, "namespace", "owner_name", "job_name", "jobInfo")

	query = `kube_job_labels * on (job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getJobMetricString(result, "namespace", "owner_name", "job_name", "jobLabel")

	query = `kube_job_spec_completions * on (job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getJobMetric(result, "namespace", "owner_name", "job_name", "specCompletions")

	query = `kube_job_spec_parallelism * on (job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getJobMetric(result, "namespace", "owner_name", "job_name", "specParallelism")

	query = `kube_job_status_completion_time * on (job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getJobMetric(result, "namespace", "owner_name", "job_name", "statusCompletionTime")

	query = `kube_job_status_start_time * on (job_name) group_left (owner_name) kube_job_owner`
	result = prometheus.MetricCollect(promaddress, query, start, end)
	getJobMetric(result, "namespace", "owner_name", "job_name", "statusStartTime")
	/*
		query = `kube_job_status_active * on (job_name) group_left (owner_name) kube_job_owner`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getJobMetric(result, "namespace", "cronjob", "job", "statusActive")

		query = `kube_job_status_failed * on (job_name) group_left (owner_name) kube_job_owner`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getJobMetric(result, "namespace", "cronjob", "job", "statusFailed")

		query = `kube_job_status_succeeded * on (job_name) group_left (owner_name) kube_job_owner`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getJobMetric(result, "namespace", "cronjob", "job", "statusSucceeded")

		query = `kube_job_complete * on (job_name) group_left (owner_name) kube_job_owner`
		result = prometheus.MetricCollect(promaddress, query, start, end)
		getJobMetric(result, "namespace", "cronjob", "job", "complete")*/

	writeConfigCronJob(clusterName, promAddr)
	writeAttributesCronJob(clusterName, promAddr)
	writeConfigJob(clusterName, promAddr)
	writeAttributesJob(clusterName, promAddr)
}
