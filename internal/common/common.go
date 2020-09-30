package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

// Parameters - Reusable structure that holds common arguments used in the project
type Parameters struct {
	ClusterName, PromURL, PromAddress, FileName, Interval *string
	IntervalSize, History, Offset                         *int
	Debug                                                 bool
	CurrentTime                                           *time.Time
	LabelSuffix                                           string
	InfoLogger, WarnLogger, ErrorLogger, DebugLogger      *log.Logger
	SampleRate                                            int
	SampleRateString                                      string
	OAuthTokenPath                                        string
	CaCertPath                                            string
}

// Prometheus Objects

//MetricCollect is used to query Prometheus to get data for specific query and return the results to be processed.
func MetricCollect(args *Parameters, query string, range5m v1.Range, metric string, vital bool) (value model.Value) {

	//setup the context to use for the API calls
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tlsClientConfig := &tls.Config{}
	if args.CaCertPath != "" {
		tmpTLSConfig, err := config.NewTLSConfig(&config.TLSConfig{
			CAFile: args.CaCertPath,
		})
		if err != nil {
			log.Fatalf("Failed to generate TLS config:%v", err)
		}
		tlsClientConfig = tmpTLSConfig
	}

	var roundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: tlsClientConfig,
	}

	if args.OAuthTokenPath != "" {
		roundTripper = config.NewBearerAuthFileRoundTripper(args.OAuthTokenPath, roundTripper)
	}
	//Setup the API client connection
	client, err := api.NewClient(api.Config{Address: *args.PromURL, RoundTripper: roundTripper})
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
func TimeRange(args *Parameters, historyInterval time.Duration) (promRange v1.Range) {

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

// AddToLabelMap used to add values to label map used for attributes.
func AddToLabelMap(key string, value string, labelPath map[string]string) {
	if _, ok := labelPath[key]; !ok {
		value = strings.Replace(value, "\n", "", -1)
		if len(value) > 255 {
			labelPath[key] = value[:255]
		} else {
			labelPath[key] = value
		}
		return
	}

	if strings.Contains(value, ";") {
		currValue := ""
		for _, l := range value {
			currValue = currValue + string(l)
			if l == ';' {
				AddToLabelMap(key, currValue[:len(currValue)-1], labelPath)
				currValue = ""
			}
		}
		AddToLabelMap(key, currValue, labelPath)
		return
	}

	currValue := ""
	notPresent := true
	for _, l := range labelPath[key] {
		currValue = currValue + string(l)
		if l == ';' {
			if currValue[:len(currValue)-1] == value {
				notPresent = false
				break
			}
			currValue = ""
		}
	}
	if currValue != value && notPresent {
		if len(value) > 255 {
			labelPath[key] = labelPath[key] + ";" + value[:255]
		} else {
			labelPath[key] = labelPath[key] + ";" + value
		}
	}
}

//GetWorkload used to query for the workload data and then calls write workload
func GetWorkload(fileName, metricName, query string, metricfield model.LabelName, args *Parameters, entityKind string) {
	var historyInterval time.Duration
	historyInterval = 0
	var result model.Value
	//Open the files that will be used for the workload data types and write out there headers.
	workloadWrite, err := os.Create("./data/" + entityKind + "/" + fileName + ".csv")
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("entity=" + entityKind + " message=" + err.Error())
		return
	}
	if entityKind == "cluster" {
		fmt.Fprintf(workloadWrite, "cluster,Datetime,%s\n", metricName)
	} else {
		fmt.Fprintf(workloadWrite, "cluster,%s,Datetime,%s\n", entityKind, metricName)
	}

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slpwer prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < *args.History; historyInterval++ {
		range5Min := TimeRange(args, historyInterval)

		result = MetricCollect(args, query, range5Min, metricName, false)
		if result != nil {
			writeWorkload(workloadWrite, result, metricfield, args, entityKind)
		}
	}
	//Close the workload files.
	workloadWrite.Close()
}

//writeWorkload will write out the workload data specific to metric provided to the file that was passed in.
func writeWorkload(file io.Writer, result model.Value, metricfield model.LabelName, args *Parameters, entityKind string) {
	//Loop through the results for the workload and validate that contains the required labels and that the entity exists in the systems data structure once validated will write out the workload for the system.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		var entity model.LabelValue
		var ok bool
		if entityKind != "cluster" {
			if entity, ok = result.(model.Matrix)[i].Metric[metricfield]; !ok {
				continue
			}
		}
		//Loop through the different values over the interval and write out each one to the workload file.
		for j := 0; j < len(result.(model.Matrix)[i].Values); j++ {
			var val model.SampleValue
			if !math.IsNaN(float64(result.(model.Matrix)[i].Values[j].Value)) && !math.IsInf(float64(result.(model.Matrix)[i].Values[j].Value), 0) {
				val = result.(model.Matrix)[i].Values[j].Value
			}
			fmt.Fprintf(file, "%s,", *args.ClusterName)
			if entityKind != "cluster" {
				fmt.Fprintf(file, "%s,", strings.Replace(string(entity), ";", ".", -1))
			}
			fmt.Fprintf(file, "%s,%f\n", time.Unix(0, int64(result.(model.Matrix)[i].Values[j].Timestamp)*1000000).Format("2006-01-02 15:04:05.000"), val)
		}
	}
}
