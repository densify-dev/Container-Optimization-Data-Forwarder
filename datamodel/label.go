package datamodel

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/common/model"
	"github.com/r3labs/diff/v2"
	"math"
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
type TimeRangeLabelsMap map[string][]*TimeRangeLabels

func (l *Labels) HasDiffs() bool {
	return len(l.Diffs) > 0
}

func (l *Labels) GetApplicableRanges() []*Range {
	var ranges []*Range
	if l.Origin != nil {
		ranges = append(ranges, l.Origin.Range)
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

type ValueConversionFunction func(float64) float64
type StringerConversionFunction func(float64) fmt.Stringer

type Converter struct {
	VCF ValueConversionFunction
	SCF StringerConversionFunction
}

func ToString(c *Converter, v model.SampleValue) string {
	var s fmt.Stringer
	if c != nil {
		f := float64(v)
		if c.VCF != nil {
			s = model.SampleValue(c.VCF(f))
		} else if c.SCF != nil {
			s = c.SCF(f)
		}
	}
	if s == nil {
		s = v
	}
	return s.String()
}

func timeConv(value float64) fmt.Stringer {
	sec, frac := math.Modf(value)
	return time.Unix(int64(sec), int64(float64(time.Second)*frac))
}

func boolConv(value float64) fmt.Stringer {
	b := value != 0.0
	return &BoolStringer{Value: b}
}

func TimeStampConverter() *Converter {
	return &Converter{SCF: timeConv}
}

func BoolConverter() *Converter {
	return &Converter{SCF: boolConv}
}

type BoolStringer struct {
	Value bool
}

func (bs *BoolStringer) String() string {
	return fmt.Sprintf("%t", bs.Value)
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
	if err == nil {
		sort.SliceStable(res, func(i, j int) bool {
			return res[i].Range.Before(res[j].Range)
		})
		n := len(res)
		for i := 0; i < n-1; i++ {
			ri := res[i].Range
			rj := res[i+1].Range
			if ri.Overlaps(rj) {
				err = multierror.Append(err, fmt.Errorf("range %s overlaps range %s", ri.String(), rj.String()))
			}
		}
	}
	return res, err
}

func AdjustRanges(trlm TimeRangeLabelsMap, d *Discovery) TimeRangeLabelsMap {
	st := make(map[int64]bool)
	for k, trls := range trlm {
		n := len(trlm)
		for i, trl := range trls {
			r := trl.Range
			r.AdjustRange(d.Range, d.MaxScrapeInterval, i == 0, i == n-1)
			// now if trl is not the last one, adjust its end and the start of the next one to the
			// median of the gap between them
			if i < n-1 {
				var nextRange *Range
				if next := trls[i+1]; next != nil {
					nextRange = next.Range
				}
				r.StretchBoth(nextRange)
			}
			// now record the stat time of the range
			st[r.Start.UnixNano()] = true
		}
		// if we are missing ranges to cover the entire discovery range, add additional TimeRangeLabels
		// with an empty map at the beginning/end of the slice
		if n > 0 {
			var modified bool
			start := trls[0].Range.Start
			if !start.Equal(*d.Range.Start) {
				rng := &Range{Start: d.Range.Start, End: add(start, -time.Nanosecond)}
				newStart := &TimeRangeLabels{Range: rng}
				trls = append([]*TimeRangeLabels{newStart}, trls...)
				st[rng.Start.UnixNano()] = true
				modified = true
			}
			end := trls[n-1].Range.End
			if !end.Equal(*d.Range.End) {
				rng := &Range{Start: add(end, time.Nanosecond), End: d.Range.End}
				newEnd := &TimeRangeLabels{Range: rng}
				trls = append(trls, newEnd)
				st[rng.Start.UnixNano()] = true
				modified = true
			}
			if modified {
				trlm[k] = trls
			}
		}
	}

	// now create a slice of start times and sort it
	n := len(st)
	startTimes := make([]int64, n)
	i := 0
	for startTime := range st {
		startTimes[i] = startTime
		i++
	}
	sort.SliceStable(startTimes, func(i, j int) bool {
		return startTimes[i] < startTimes[j]
	})
	actualRanges := make([]*Range, n)
	for i, startTime := range startTimes {
		t := time.Unix(0, startTime)
		actualRanges[i] = &Range{Start: &t, End: &t}
	}
	for i, ar := range actualRanges {
		if i == n-1 {
			ar.AdjustRange(d.Range, d.MaxScrapeInterval, false, true)
		} else {
			ar.StretchTo(actualRanges[i+1])
		}
	}
	res := make(TimeRangeLabelsMap, len(trlm))
	for k, orgTrls := range trlm {
		trls := make([]*TimeRangeLabels, n)
		for i, ar := range actualRanges {
			for _, trl := range orgTrls {
				if trl.Range.Contains(ar) {
					trls[i] = &TimeRangeLabels{Map: trl.Map, Range: ar}
				}
				break
			}
		}
		res[k] = trls
	}
	return res
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
