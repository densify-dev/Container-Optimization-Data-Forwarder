package datamodel

import (
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/r3labs/diff/v2"
	"sort"
	"time"
)

type Diff struct {
	Changelog diff.Changelog `json:"changelog,omitempty"`
	Range     *Range         `json:"range,omitempty"`
}

type Labels struct {
	Original  *RangeLabels      `json:"original,omitempty"`
	Diffs     []*Diff           `json:"diffs,omitempty"`
	currMap   map[string]string // unexported, helper field when building or converting
	currRange *Range            // same
}

type LabelMap map[string]*Labels

func (l *Labels) HasDiffs() bool {
	return len(l.Diffs) > 0
}

func (l *Labels) GetApplicableRanges() []*Range {
	var ranges []*Range
	if l.Original != nil {
		ranges = append(ranges, l.Original.Range)
	}
	for _, d := range l.Diffs {
		ranges = append(ranges, d.Range)
	}
	return ranges
}

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
	if l.Original == nil {
		// uninitialized labels
		l.Original = &RangeLabels{Map: m, Range: r}
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

func (l *Labels) AppendSampleStreamWithFilter(ss *model.SampleStream, filter []string) error {
	if m, err := filterMetric(ss.Metric, filter); err == nil {
		fss := &model.SampleStream{Metric: m, Values: ss.Values}
		return l.AppendSampleStream(fss)
	} else {
		return err
	}
}

func (l *Labels) AppendSampleWithFilter(s *model.Sample, filter []string) error {
	if m, err := filterMetric(s.Metric, filter); err == nil {
		fs := &model.Sample{Metric: m, Value: s.Value, Timestamp: s.Timestamp}
		return l.AppendSample(fs)
	} else {
		return err
	}
}

func (l *Labels) AppendSampleStreamWithValue(ss *model.SampleStream, key string, c *Converter) error {
	for _, sp := range ss.Values {
		t := sp.Timestamp.Time()
		if err := l.appendValue(ss.Metric, key, sp.Value, c, &t); err != nil {
			return err
		}
	}
	return nil
}

func (l *Labels) AppendSampleWithValue(s *model.Sample, key string, c *Converter) error {
	t := s.Timestamp.Time()
	return l.appendValue(s.Metric, key, s.Value, c, &t)
}

func EnsureLabels(lm LabelMap, key string) *Labels {
	var labels *Labels
	var f bool
	if labels, f = lm[key]; !f {
		labels = &Labels{}
		lm[key] = labels
	}
	return labels
}

func (l *Labels) append(met model.Metric, ts []*time.Time) error {
	m := make(map[string]string, len(met))
	for ln, lv := range met {
		m[string(ln)] = string(lv)
	}
	return l.AppendMap(m, ts)
}

func (l *Labels) appendValue(met model.Metric, key string, v model.SampleValue, c *Converter, t *time.Time) error {
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
	return l.AppendMap(map[string]string{actualKey: ToString(c, v)}, []*time.Time{t})
}

func filterMetric(met model.Metric, f []string) (model.Metric, error) {
	n := len(f)
	if n == 0 {
		return nil, fmt.Errorf("empty filter provided")
	}
	m := make(model.Metric, n)
	for _, key := range f {
		ln := model.LabelName(key)
		if lv, ok := met[ln]; ok {
			m[ln] = lv
		} else {
			return nil, fmt.Errorf("no value for key %s", key)
		}
	}
	return m, nil
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
