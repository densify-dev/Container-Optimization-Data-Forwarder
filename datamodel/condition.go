package datamodel

import (
	"fmt"
	"github.com/prometheus/common/model"
	"strings"
)

const (
	ConditionKey = "condition"
	StatusKey    = "status"
	// condition status values, copied from k8s.io/api package to avoid dependencies
	ConditionTrue    = "True"
	ConditionFalse   = "False"
	ConditionUnknown = "Unknown"
	// Float values
	FloatTrue    = 1.0
	FloatFalse   = 0.0
	FloatUnknown = -1.0
)

var cm = map[string]float64{
	strings.ToLower(ConditionTrue):    FloatTrue,
	strings.ToLower(ConditionFalse):   FloatFalse,
	strings.ToLower(ConditionUnknown): FloatUnknown,
}

var rcm = map[float64]string{
	FloatTrue:    ConditionTrue,
	FloatFalse:   ConditionFalse,
	FloatUnknown: ConditionUnknown,
}

var noConditionValues = len(cm)

func ConditionFloat(s string) (v float64, f bool) {
	v, f = cm[strings.ToLower(s)]
	return
}

func ConditionString(v float64) (s string, f bool) {
	s, f = rcm[v]
	return
}

type StatusMap map[string][]model.SamplePair

func (sm StatusMap) IsComplete() bool {
	return len(sm) == noConditionValues
}

type Condition struct {
	metric    model.Metric
	statusMap StatusMap
}

func (c *Condition) IsComplete() bool {
	return c.statusMap.IsComplete()
}

func (c *Condition) Append(ss *model.SampleStream) error {
	var status string
	if sts, ok := ss.Metric[StatusKey]; ok {
		status = string(sts)
	} else {
		return fmt.Errorf("%s value not found", StatusKey)
	}
	// can copy Metric over
	if len(c.metric) == 0 {
		c.metric = ss.Metric.Clone()
		delete(c.metric, StatusKey)
	}
	if c.statusMap == nil {
		c.statusMap = make(StatusMap)
	}
	c.statusMap[status] = ss.Values
	return nil
}

func (c *Condition) Consolidate() (ss *model.SampleStream, err error) {
	if c.IsComplete() {
		ss = &model.SampleStream{Metric: c.metric}
		l := -1
		cond := c.metric[model.LabelName(ConditionKey)]
		for status, values := range c.statusMap {
			var statusValue float64
			var f bool
			if statusValue, f = ConditionFloat(status); !f {
				err = fmt.Errorf("invalid status value %s", status)
			}
			if err == nil {
				if l == -1 {
					l = len(values)
				} else if l1 := len(values); l != l1 {
					err = fmt.Errorf("different status map lengths for %v: %d, %d", cond, l, l1)
				}
			}
			if err == nil {
				for _, value := range values {
					if ToBool(float64(value.Value)) {
						newValue := model.SamplePair{
							Timestamp: value.Timestamp,
							Value:     model.SampleValue(statusValue),
						}
						ss.Values = append(ss.Values, newValue)
					}
				}
			}
		}
		// check that all timestamps have been encountered for
		if lf := len(ss.Values); lf == l {
			// need to sort the values
			SortSampleStream(ss)
		} else {
			err = fmt.Errorf("bad reconsntructed sample stream for %v: expected %d, got %d", cond, l, lf)
		}
	} else {
		err = fmt.Errorf("condition is not complete")
	}
	if err != nil {
		ss = nil
	}
	return
}
