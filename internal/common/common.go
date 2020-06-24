package common

import (
	"log"
	"strings"
	"time"
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
