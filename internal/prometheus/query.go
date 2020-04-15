//Package prometheus collects data from Prometheus and returns the results.
package prometheus

import (
	"context"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/logger"
	//"github.com/densify-dev/Container-Optimization-Data-Forwarder/cmd/datacollection"
	"github.com/prometheus/common/model"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// CollectionArgs -
type CollectionArgs struct {
	PromURL, Query, EntityKind *string
	Range                      *v1.Range
}

var promAddLog string
var hasClusterName = false

//MetricCollect is used to query Prometheus to get data for specific query and return the results to be processed.
func MetricCollect(args *CollectionArgs, metric string, vital bool) (value model.Value, logLine string) {

	//setup the context to use for the API calls
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Setup the API client connection
	client, err := api.NewClient(api.Config{Address: *args.PromURL})
	if err != nil {
		return value, logger.LogError(map[string]string{"message": err.Error(), "query": *args.Query, "metric": metric}, "WARN")
	}

	//Query prometheus with the values defined above as well as the query that was passed into the function.
	q := v1.NewAPI(client)
	value, _, err = q.QueryRange(ctx, *args.Query, *args.Range)
	if err != nil {
		return value, logger.LogError(map[string]string{"message": err.Error(), "query": *args.Query, "metric": metric}, "ERROR")
	}

	//If the values from the query return no data (length of 0) then give a warning
	if value == nil {
		if vital {
			return value, logger.LogError(map[string]string{"message": "No data returned from value", "query": *args.Query, "metric": metric}, "ERROR")
		}
		return value, logger.LogError(map[string]string{"message": "No data returned", "query": *args.Query, "metric": metric}, "WARN")
	} else if value.(model.Matrix) == nil {
		if vital {
			return value, logger.LogError(map[string]string{"message": "No data returned", "query": *args.Query, "metric": metric}, "ERROR")
		}
		return value, logger.LogError(map[string]string{"message": "No data returned", "query": *args.Query, "metric": metric}, "WARN")
	} else if value.(model.Matrix).Len() == 0 {
		if vital {
			return value, logger.LogError(map[string]string{"message": "No data returned, value.(model.Matrix) is empty", "query": *args.Query, "metric": metric}, "ERROR")
		}
		return value, logger.LogError(map[string]string{"message": "No data returned", "query": *args.Query, "metric": metric}, "WARN")
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
