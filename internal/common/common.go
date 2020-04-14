package common

import "time"

// Parameters - Reusable structure that holds common arguments used in the project
type Parameters struct {
	ClusterName, PromURL, PromAddress, FileName, Interval *string
	IntervalSize, History                                 *int
	Debug                                                 bool
	CurrentTime                                           *time.Time
	LabelSuffix                                           string
}
