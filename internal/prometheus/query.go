//Package prometheus collects data from Prometheus and returns the results.
package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/densify-dev/Container-Optimization-Data-Forwarder/internal/common"

	"github.com/prometheus/common/model"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

//MetricCollect is used to query Prometheus to get data for specific query and return the results to be processed.
func MetricCollect(args *common.Parameters, query string, range5m v1.Range, metric string, vital bool) (value model.Value) {

	//setup the context to use for the API calls
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Setup the API client connection
	client, err := api.NewClient(api.Config{Address: *args.PromURL})
	if err != nil {
		args.WarnLogger.Println("metric=" + metric + " query=" + query + " message=" + err.Error())
		fmt.Println("metric=" + metric + " query=" + query + " message=" + err.Error())
		return value
	}

	//Query prometheus with the values defined above as well as the query that was passed into the function.
	q := v1.NewAPI(client)
	value, _, err = q.QueryRange(ctx, query, range5m)
	if err != nil {
		args.ErrorLogger.Println("metric=" + metric + " query=" + query + " message=" + err.Error())
		fmt.Println("metric=" + metric + " query=" + query + " message=" + err.Error())
		return value
	}

	//If the values from the query return no data (length of 0) then give a warning
	if value == nil {
		if vital {
			args.ErrorLogger.Println("metric=" + metric + " query=" + query + " message=No resultset returned")
			fmt.Println("metric=" + metric + " query=" + query + " message=No resultset returned")
			return value
		}
		args.WarnLogger.Println("metric=" + metric + " query=" + query + " message=No resultset returned")
		fmt.Println("metric=" + metric + " query=" + query + " message=No resultset returned")
		return value

	} else if value.(model.Matrix) == nil {
		if vital {
			args.ErrorLogger.Println("metric=" + metric + " query=" + query + " message=No time series data returned")
			fmt.Println("metric=" + metric + " query=" + query + " message=No time series data returned")
			return value
		}
		args.WarnLogger.Println("metric=" + metric + " query=" + query + " message=No time series data returned")
		fmt.Println("metric=" + metric + " query=" + query + " message=No time series data returned")
		return value
	} else if value.(model.Matrix).Len() == 0 {
		if vital {
			args.ErrorLogger.Println("metric=" + metric + " query=" + query + " message=No data returned, value.(model.Matrix) is empty")
			fmt.Println("metric=" + metric + " query=" + query + " message=No data returned, value.(model.Matrix) is empty")
			return value
		}
		args.WarnLogger.Println("metric=" + metric + " query=" + query + " message=No data returned, value.(model.Matrix) is empty")
		fmt.Println("metric=" + metric + " query=" + query + " message=No data returned, value.(model.Matrix) is empty")

	}

	//Return the data that was received from Prometheus.
	return value
}

//TimeRange allows you to define the start and end values of the range will pass to the Prometheus for the query.
func TimeRange(args *common.Parameters, historyInterval time.Duration) (promRange v1.Range) {

	var start, end time.Time

	//For workload metrics the historyInterval will be set depending on how far back in history we are querying currently. Note it will be 0 for all queries that are not workload related.
	if *args.Interval == "days" {
		start = args.CurrentTime.Add(time.Hour * -24 * time.Duration(*args.IntervalSize)).Add(time.Hour * -24 * time.Duration(*args.IntervalSize) * historyInterval)
		end = args.CurrentTime.Add(time.Hour * -24 * time.Duration(*args.IntervalSize) * historyInterval)
	} else if *args.Interval == "hours" {
		start = args.CurrentTime.Add(time.Hour * -1 * time.Duration(*args.IntervalSize)).Add(time.Hour * -1 * time.Duration(*args.IntervalSize) * historyInterval)
		end = args.CurrentTime.Add(time.Hour * -1 * time.Duration(*args.IntervalSize) * historyInterval)
	} else {
		start = args.CurrentTime.Add(time.Minute * -1 * time.Duration(*args.IntervalSize)).Add(time.Minute * -1 * time.Duration(*args.IntervalSize) * historyInterval)
		end = args.CurrentTime.Add(time.Minute * -1 * time.Duration(*args.IntervalSize) * historyInterval)
	}

	return v1.Range{Start: start, End: end, Step: time.Minute * time.Duration(args.SampleRate)}
}
