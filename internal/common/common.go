package common

import "time"

// ARGS - Reusable structure that holds common arguments used in the project
type ARGS struct {
	ClusterName, PromURL, PromAddress, FileName, MetricName, Query, Interval, Prefix, Metric *string
	IntervalSize, History                                                                    *int
	Debug                                                                                    bool
	CurrentTime                                                                              *time.Time
}
