package datamodel

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/common/model"
	"github.com/r3labs/diff/v2"
	"sort"
	"time"
)

type TimeRangeLabels struct {
	Map   map[string]string `json:"map,omitempty"`
	Range *Range            `json:"range,omitempty"`
}

type Diff struct {
	Changelog diff.Changelog `json:"changelog,omitempty"`
	Range     *Range         `json:"range,omitempty"`
}

type Labels struct {
	Origin    *TimeRangeLabels  `json:"origin,omitempty"`
	Diffs     []*Diff           `json:"diffs,omitempty"`
	currMap   map[string]string // unexported, helper field when building or converting
	currRange *Range            // same
}

type LabelMap map[string]*Labels

func (l *Labels) AppendMap(m map[string]string, ts []*time.Time) error {
	var start, end *time.Time
	if n := len(ts); n == 0 {
		return fmt.Errorf("append map called with no time series")
	} else {
		sort.SliceStable(ts, func(i, j int) bool { return ts[i].Before(*ts[j]) })
		start = ts[0]
		end = ts[n-1]
	}
	r := &Range{Start: start, End: end}
	var err error
	if l.Origin == nil {
		// uninitialized labels
		l.Origin = &TimeRangeLabels{Map: m, Range: r}
		l.setCurrents(m, r)
	} else {
		var cl diff.Changelog
		if cl, err = diff.Diff(l.currMap, m); err == nil {
			if len(cl) == 0 {
				// just need to potentially expand the current range with the one we've computed
				l.currRange.ExpandRange(r)
			} else {
				l.Diffs = append(l.Diffs, &Diff{Changelog: cl, Range: r})
				l.setCurrents(m, r)
			}
		}
	}
	return err
}

func (l *Labels) AppendSampleStream(ss *model.SampleStream) error {
	var ts []*time.Time
	for _, sp := range ss.Values {
		t := sp.Timestamp.Time()
		ts = append(ts, &t)
	}
	return l.append(ss.Metric, ts)
}

func (l *Labels) AppendSample(s *model.Sample) error {
	t := s.Timestamp.Time()
	return l.append(s.Metric, []*time.Time{&t})
}

type ValueConversionFunction func(float64) float64

func (l *Labels) AppendSampleStreamWithValue(ss *model.SampleStream, key string, vcf ValueConversionFunction) error {
	for _, sp := range ss.Values {
		t := sp.Timestamp.Time()
		if err := l.appendValue(ss.Metric, key, sp.Value, vcf, &t); err != nil {
			return err
		}
	}
	return nil
}

func (l *Labels) AppendSampleWithValue(s *model.Sample, key string, vcf ValueConversionFunction) error {
	t := s.Timestamp.Time()
	return l.appendValue(s.Metric, key, s.Value, vcf, &t)
}

func (l *Labels) ToTimeRangeLabels() ([]*TimeRangeLabels, error) {
	var res []*TimeRangeLabels
	var err error
	if l.Origin != nil {
		res = append(res, l.Origin)
		l.setCurrents(l.Origin.Map, l.Origin.Range)
		for _, d := range l.Diffs {
			if d != nil {
				m := copyMap(l.currMap)
				pl := diff.Patch(d.Changelog, m)
				for _, ple := range pl {
					if ple.Errors != nil {
						err = multierror.Append(err, ple.Errors)
					}
				}
				if err == nil {
					res = append(res, &TimeRangeLabels{Map: m, Range: d.Range})
					l.setCurrents(m, d.Range)
				} else {
					break
				}
			}
		}
	}
	return res, err
}

func (l *Labels) append(met model.Metric, ts []*time.Time) error {
	m := make(map[string]string, len(met))
	for ln, lv := range met {
		m[string(ln)] = string(lv)
	}
	return l.AppendMap(m, ts)
}

func (l *Labels) appendValue(met model.Metric, key string, v model.SampleValue, vcf ValueConversionFunction, t *time.Time) error {
	var actualKey string
	if key == "" {
		actualKey = SingleValueKey
	} else {
		if ak, ok := met[model.LabelName(key)]; ok {
			actualKey = string(ak)
		} else {
			return fmt.Errorf("key %s not found in labelset %v", key, met)
		}
	}
	var val model.SampleValue
	if vcf == nil {
		val = v
	} else {
		val = model.SampleValue(vcf(float64(v)))
	}
	return l.AppendMap(map[string]string{actualKey: val.String()}, []*time.Time{t})
}

func (l *Labels) setCurrents(m map[string]string, r *Range) {
	l.currMap = m
	l.currRange = r
}

func copyMap(m map[string]string) map[string]string {
	r := make(map[string]string, len(m))
	for k, v := range m {
		r[k] = v
	}
	return r
}
