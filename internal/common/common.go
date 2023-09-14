package common

import (
	"context"
	"crypto/tls"
	"errors"
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
	SampleRateString, NodeGroupList                       string
	OAuthTokenPath                                        string
	CaCertPath                                            string
	Deployments, CronJobs                                 bool
}

// Prometheus Objects

// MetricCollect is used to query Prometheus to get data for specific query and return the results to be processed.
func MetricCollect(args *Parameters, query string, range5m v1.Range) (value model.Value, err error) {
	var pa v1.API
	if args.Debug {
		// range5m is always the same, no point in logging
		msg := fmt.Sprintf("QueryRange: query = %s", query)
		args.DebugLogger.Println(msg)
		fmt.Println("[DEBUG] " + msg)
	}
	ctx, cancel := context.WithCancel(context.Background())
	_ = time.AfterFunc(2*time.Minute, func() { cancel() })
	if pa, err = promApi(args); err != nil {
		return
	}
	if value, _, err = pa.QueryRange(ctx, query, range5m); err != nil {
		return
	}
	if value == nil {
		err = errors.New("no resultset returned")
	} else if value.(model.Matrix) == nil {
		err = errors.New("no time series data returned")
	} else if value.(model.Matrix).Len() == 0 {
		err = errors.New("no data returned, value.(model.Matrix) is empty")
	}
	return
}

func GetVersion(args *Parameters) (version string, err error) {
	var pa v1.API
	ctx, cancel := context.WithCancel(context.Background())
	_ = time.AfterFunc(30*time.Second, func() { cancel() })
	if pa, err = promApi(args); err != nil {
		return
	}
	var bir v1.BuildinfoResult
	if bir, err = pa.Buildinfo(ctx); err == nil {
		version = bir.Version
	}
	return
}

func promApi(args *Parameters) (v1.API, error) {
	tlsClientConfig := &tls.Config{}
	if args.CaCertPath != "" {
		if c, err := config.NewTLSConfig(&config.TLSConfig{
			CAFile: args.CaCertPath,
		}); err == nil {
			tlsClientConfig = c
		} else {
			log.Fatalf("Failed to generate TLS config:%v", err)
		}
	}
	var roundTripper http.RoundTripper = &http.Transport{
		TLSClientConfig: tlsClientConfig,
	}
	if args.OAuthTokenPath != "" {
		roundTripper = config.NewAuthorizationCredentialsFileRoundTripper("Bearer", args.OAuthTokenPath, roundTripper)
	}
	if client, err := api.NewClient(api.Config{Address: *args.PromURL, RoundTripper: roundTripper}); err == nil {
		return v1.NewAPI(client), nil
	} else {
		return nil, err
	}
}

// TimeRange allows you to define the start and end values of the range will pass to the Prometheus for the query.
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
		value = strings.Replace(value, "\r", "", -1)
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

// GetWorkload used to query for the workload data and then calls write workload
func GetWorkload(fileName, metricName, query string, metricField []model.LabelName, args *Parameters, entityKind string) {
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
	if csvHeaderFormat, f := GetCsvHeaderFormat(entityKind); f {
		fmt.Fprintf(workloadWrite, csvHeaderFormat, metricName)
	} else {
		msg := " message=no CSV header format found"
		args.ErrorLogger.Println("entity=" + entityKind + msg)
		fmt.Println("entity=" + entityKind + msg)
		return
	}

	//If the History parameter is set to anything but default 1 then will loop through the calls starting with the current day\hour\minute interval and work backwards.
	//This is done as the farther you go back in time the slower prometheus querying becomes and we have seen cases where will not run from timeouts on Prometheus.
	//As a result if we do hit an issue with timing out on Prometheus side we still can send the current data and data going back to that point vs losing it all.
	for historyInterval = 0; int(historyInterval) < *args.History; historyInterval++ {
		range5Min := TimeRange(args, historyInterval)

		result, err = MetricCollect(args, query, range5Min)
		if err != nil {
			args.WarnLogger.Println("metric=" + metricName + " query=" + query + " message=" + err.Error())
			fmt.Println("[WARNING] metric=" + metricName + " query=" + query + " message=" + err.Error())
		} else {
			WriteWorkload(workloadWrite, result, metricField, args, entityKind)
		}
	}
	//Close the workload files.
	workloadWrite.Close()
}

// WriteWorkload will write out the workload data specific to metric provided to the file that was passed in.
func WriteWorkload(file io.Writer, result model.Value, metricField []model.LabelName, args *Parameters, entityKind string) {
	//Loop through the results for the workload and validate that contains the required labels and that the entity exists in the systems data structure once validated will write out the workload for the system.
	for i := 0; i < result.(model.Matrix).Len(); i++ {
		var field, field2 model.LabelValue
		var ok bool
		if entityKind != "cluster" {
			if field, ok = result.(model.Matrix)[i].Metric[metricField[0]]; !ok {
				continue
			}
		}
		if entityKind == "rq" {
			if field2, ok = result.(model.Matrix)[i].Metric[metricField[1]]; !ok {
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
				fmt.Fprintf(file, "%s,", strings.Replace(string(field), ";", ".", -1))
			}
			if entityKind == "rq" {
				fmt.Fprintf(file, "%s,", strings.Replace(string(field2), ";", ".", -1))
			}
			fmt.Fprintf(file, "%s,%f\n", FormatTime(result.(model.Matrix)[i].Values[j].Timestamp), val)
		}
	}
}

func FormatTime(mt model.Time) string {
	t := mt.Time()
	return Format(&t)
}

func FormatTimeInSec(i int64) string {
	t := time.Unix(i, 0)
	return Format(&t)
}

func Format(t *time.Time) string {
	return t.Format(time.RFC3339Nano)
}

func GetCsvHeaderFormat(entityKind string) (string, bool) {
	ek := strings.ToLower(entityKind)
	format, f := csvHeaderFormats[ek]
	return format, f
}

type headerBuilder struct {
	entityKindName     string
	includeClusterName bool
	includeNamespace   bool
}

const (
	containerEntityKindName    = "EntityName,EntityType,ContainerName"
	containerHpaEntityKindName = containerEntityKindName + ",HpaName"
)

var headerBuilders = map[string]*headerBuilder{
	"cluster":       {entityKindName: "Name"},
	"node":          {entityKindName: "NodeName", includeClusterName: true},
	"node_group":    {entityKindName: "NodeGroupName", includeClusterName: true},
	"rq":            {entityKindName: "RqName", includeClusterName: true, includeNamespace: true},
	"crq":           {entityKindName: "CrqName", includeClusterName: true},
	"container":     {entityKindName: containerEntityKindName, includeClusterName: true, includeNamespace: true},
	"container_hpa": {entityKindName: containerHpaEntityKindName, includeClusterName: true, includeNamespace: true},
}

func (hb *headerBuilder) generateCsvHeaderFormat() string {
	l := 3
	if hb.includeClusterName {
		l++
	}
	if hb.includeNamespace {
		l++
	}
	components := make([]string, l)
	if hb.includeClusterName {
		components[0] = "ClusterName"
	}
	if hb.includeNamespace {
		components[1] = "Namespace"
	}
	components[l-3] = hb.entityKindName
	components[l-2] = "MetricTime"
	components[l-1] = "%s\n"
	return strings.Join(components, ",")
}

var csvHeaderFormats = makeCsvHeaderFormats()

func makeCsvHeaderFormats() map[string]string {
	m := make(map[string]string, len(headerBuilders))
	for entityKind, hb := range headerBuilders {
		m[entityKind] = hb.generateCsvHeaderFormat()
	}
	return m
}
