package prometheus

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

const (
	// Entity Kinds
	ContainerEntityKind = "container"
	NodeEntityKind      = "node"
	RQEntityKind        = "rq"
	CRQEntityKind       = "crq"
)

var NamespaceFilter = []string{datamodel.NamespaceKey}

// Parameters - Reusable structure that holds common arguments used in the project
type Parameters struct {
	ClusterName, PromURL                             *string
	Debug                                            bool
	InfoLogger, WarnLogger, ErrorLogger, DebugLogger *log.Logger
	OAuthTokenPath                                   string
	CaCertPath                                       string
	CurrentTime                                      *time.Time
	StartTime                                        *time.Time
	History                                          string
	StartTimeInclusive                               bool
	Range5Min                                        v1.Range
	Version                                          string
}

func (args *Parameters) ToDiscovery(entityKind string) (*datamodel.Discovery, error) {
	var disc *datamodel.Discovery
	var err error
	if maxScrapeInterval, ok := entityMaxScrapeIntervals[entityKind]; ok {
		start := *args.StartTime
		if args.StartTimeInclusive {
			start = start.Add(datamodel.PrometheusGranularity)
		}
		r := &datamodel.Range{Start: &start, End: args.CurrentTime}
		disc = &datamodel.Discovery{
			ClusterName:       *args.ClusterName,
			Range:             r,
			MaxScrapeInterval: maxScrapeInterval,
		}
	} else {
		err = fmt.Errorf("unknown entity kind %s", entityKind)
	}
	return disc, err
}

var promApi *promAPIv2
var apiMu sync.Mutex

// MetricCollect is used to query Prometheus to get data for specific query and return the results to be processed.
func MetricCollect(args *Parameters, query string) (value model.Value, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err = ensureApi(args); err != nil {
		return
	}
	// use Query API, not QueryRange for both discovery and workload
	query = fmt.Sprintf("%s[%s]", query, args.History)
	if value, _, err = promApi.Query(ctx, query, *args.CurrentTime); err != nil {
		return
	}
	// if the values from the query return no data (length of 0) then give a warning
	if v, ok := value.(model.Matrix); !ok {
		err = errors.New("value is nil or not a model.Matrix")
	} else if n := v.Len(); n == 0 {
		err = errors.New("no data returned, matrix is empty")
	} else {
		// make sure that values in sample stream are sorted
		for _, ss := range v {
			sort.SliceStable(ss.Values, func(i, j int) bool {
				return ss.Values[i].Timestamp.Before(ss.Values[j].Timestamp)
			})
			// filter out the first value iff its time is the start time
			if args.StartTimeInclusive {
				if t := ss.Values[0].Timestamp.Time(); t.Equal(*args.StartTime) {
					// truncate the first element
					if n == 1 {
						ss.Values = nil
					} else {
						ss.Values = ss.Values[1:]
					}
				}
			}
		}
	}
	return
}

//GetWorkload used to query for the workload data and then calls write workload
func GetWorkload(fileName, query string, args *Parameters, entityKind string) {
	GetWorkloadRelabel(fileName, query, args, entityKind, nil)
}

type ValueConversionFunc func(string) (string, bool)

type RelabelArgs struct {
	Key string
	Map map[string]string
	VCF ValueConversionFunc
}

func GetWorkloadRelabel(fileName, query string, args *Parameters, entityKind string, ras ...*RelabelArgs) {
	result, err := MetricCollect(args, query)
	if err != nil {
		args.WarnLogger.Println("metric=" + fileName + " query=" + query + " message=" + err.Error())
		fmt.Println("[WARNING] metric=" + fileName + " query=" + query + " message=" + err.Error())
	} else {
		relabel(result, ras...)
		file, _ := json.Marshal(result)
		err = os.WriteFile("./data/"+entityKind+"/"+fileName+".json", file, 0644)
		if err != nil {
			args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
			fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
		}
	}
}

func relabel(result model.Value, ras ...*RelabelArgs) {
	if mat, ok := result.(model.Matrix); ok {
		for _, ra := range ras {
			if ra != nil {
				for _, ss := range mat {
					key := model.LabelName(ra.Key)
					if val, f := ss.Metric[key]; f {
						value := string(val)
						if ra.VCF != nil {
							if v, converted := ra.VCF(value); converted {
								value = v
							}
						}
						if replacement, found := ra.Map[value]; found {
							ss.Metric[key] = model.LabelValue(replacement)
						}
					}
				}
			}
		}
	}
}

//WriteDiscovery will create the attributes.csv file that is will be sent to Densify by the Forwarder.
func WriteDiscovery(args *Parameters, entityInterface interface{}, entityKind string) {
	//Create the discovery file.
	file, _ := json.Marshal(entityInterface)
	err := os.WriteFile("./data/"+entityKind+"/discovery.json", file, 0644)
	if err != nil {
		args.ErrorLogger.Println("entity=" + entityKind + " message=" + err.Error())
		fmt.Println("[ERROR] entity=" + entityKind + " message=" + err.Error())
	}
}

// Prometheus exporters
const (
	ne   = "node-exporter"
	cad  = "cadvisor"
	ksm  = "kube-state-metrics"
	ossm = "openshift-state-metrics"
)

var exporters = []string{ne, cad, ksm, ossm}

// observed scrape intervals
// k8s cluster 	ne 	cad 	ksm 	ossm
// ------------ --- ------- ------- ------
// bare-metal	15s 25-46s	30s		N/A
// EKS 			60s 60s		60s		N/A
// GKE 			5s  12-25s	5s		N/A
// AKS 			30s 17-40s  30s 	N/A
// CRC 			15s 30s 	60s 	120s
// ------------ --- ------- ------- ------
// max + margin 75s 75s 	75s 	150s

var maxScrapeIntervals = map[string]time.Duration{
	ne:   time.Second * 75,
	cad:  time.Second * 75,
	ksm:  time.Second * 75,
	ossm: time.Second * 150,
}

var entityExporterMap = map[string][]string{
	ContainerEntityKind: {ksm, cad},
	NodeEntityKind:      {ksm, ne},
	RQEntityKind:        {ksm},
	CRQEntityKind:       {ossm},
}

var entityKinds = []string{ContainerEntityKind, NodeEntityKind, RQEntityKind, CRQEntityKind}

var entityMaxScrapeIntervals = initMaxScrapeIntervalMap()

func initMaxScrapeIntervalMap() map[string]*time.Duration {
	m := make(map[string]*time.Duration, len(entityKinds))
	for _, entityKind := range entityKinds {
		var maxInterval time.Duration
		exporters := entityExporterMap[entityKind]
		for _, exporter := range exporters {
			d := maxScrapeIntervals[exporter]
			if d > maxInterval {
				maxInterval = d
			}
		}
		m[entityKind] = &maxInterval
	}
	return m
}

func ensureApi(args *Parameters) error {
	apiMu.Lock()
	defer apiMu.Unlock()
	if promApi != nil {
		return nil
	}
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
		roundTripper = config.NewAuthorizationCredentialsFileRoundTripper("Bearer", args.OAuthTokenPath, roundTripper)
	}
	client, err := api.NewClient(api.Config{Address: *args.PromURL, RoundTripper: roundTripper})
	if err != nil {
		return err
	}
	a := v1.NewAPI(client)
	promApi = &promAPIv2{API: a, c: client}
	return nil
}
