//Package prometheus collects data from Prometheus and returns the results.
package prometheus

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/logger"
	//"github.com/densify-dev/Container-Optimization-Data-Forwarder/cmd/datacollection"
	"github.com/prometheus/common/model"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

//step is set to be 5minutes as it is defined in microseconds.
const step = 300000000000

var promAddLog string
var hasClusterName = false

//MetricCollect is used to query Prometheus to get data for specific query and return the results to be processed.
func MetricCollect(promaddress, query string, start, end time.Time, entityKind, metric string, vital bool) (value model.Value, logLine string) {

	//setup the context to use for the API calls
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Setup the API client connection
	client, err := api.NewClient(api.Config{Address: promaddress})
	if err != nil {
		return value, logger.LogError(map[string]string{"message": err.Error(), "query": query, "metric": metric}, "WARN")
	}

	//Query prometheus with the values defined above as well as the query that was passed into the function.
	q := v1.NewAPI(client)
	value, _, err = q.QueryRange(ctx, query, v1.Range{Start: start, End: end, Step: step})
	if err != nil {
		return value, logger.LogError(map[string]string{"message": err.Error(), "query": query, "metric": metric}, "ERROR")
	}

	//If the values from the query return no data (length of 0) then give a warning
	if value == nil {
		if vital {
			return value, logger.LogError(map[string]string{"message": "No data returned from value", "query": query, "metric": metric}, "ERROR")
		}
		return value, logger.LogError(map[string]string{"message": "No data returned", "query": query, "metric": metric}, "WARN")
	} else if value.(model.Matrix) == nil {
		if vital {
			return value, logger.LogError(map[string]string{"message": "No data returned", "query": query, "metric": metric}, "ERROR")
		}
		return value, logger.LogError(map[string]string{"message": "No data returned", "query": query, "metric": metric}, "WARN")
	} else if value.(model.Matrix).Len() == 0 {
		if vital {
			return value, logger.LogError(map[string]string{"message": "No data returned, value.(model.Matrix) is empty", "query": query, "metric": metric}, "ERROR")
		}
		return value, logger.LogError(map[string]string{"message": "No data returned", "query": query, "metric": metric}, "WARN")
	}

	//Return the data that was received from Prometheus.
	return value, ""
}

//TimeRange allows you to define the start and end values of the range will pass to the Prometheus for the query.
func TimeRange(interval string, intervalSize int, currentTime time.Time, historyInterval time.Duration) (start, end time.Time) {
	//define the start and end times to be used for querying prometheus based on the time the script called.
	//Depending on the Interval and interval size will determine the start and end times.
	//For workload metrics the historyInterval will be set depending on how far back in history we are querying currently. Note it will be 0 for all queries that are not workload related.
	if interval == "days" {
		start = currentTime.Add(time.Hour * -24 * time.Duration(intervalSize)).Add(time.Hour * -24 * time.Duration(intervalSize) * historyInterval)
		end = currentTime.Add(time.Hour * -24 * time.Duration(intervalSize) * historyInterval)
	} else if interval == "hours" {
		start = currentTime.Add(time.Hour * -1 * time.Duration(intervalSize)).Add(time.Hour * -1 * time.Duration(intervalSize) * historyInterval)
		end = currentTime.Add(time.Hour * -1 * time.Duration(intervalSize) * historyInterval)
	} else {
		start = currentTime.Add(time.Minute * -1 * time.Duration(intervalSize)).Add(time.Minute * -1 * time.Duration(intervalSize) * historyInterval)
		end = currentTime.Add(time.Minute * -1 * time.Duration(intervalSize) * historyInterval)
	}
	return start, end
}

//LogMessage formats and logs errors, warnings and debug messages
func LogMessage(logType, promA, entityKind, metric, message, query string) string {
	//Checks to see if the cluster name for log printing has been made. If not then run
	if hasClusterName == false {
		//Cuts everthing off before the ://
		delimiter := "://"
		rightOf := strings.Join(strings.Split(promA, delimiter)[1:], delimiter)

		//Removes the colon and port number
		delimiter1 := regexp.MustCompile(`[:]\d+`)
		promAddLog = delimiter1.ReplaceAllString(rightOf, "")

		//Sets to true to ensure this is not run again when cluster name is aqquired
		hasClusterName = true
	}
	return logType + " address=" + promAddLog + " " + "entity=" + `"` + entityKind + ", " + metric + `"` + " " + "message=" + `"` + message + `"` + " " + "query=" + `"` + query + `"`
}
