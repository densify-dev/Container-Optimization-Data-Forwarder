package datamodel

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/r3labs/diff/v2"
	"sort"
	"time"
)

type TimeRangeLabels struct {
	Map   map[string]string `json:"map,omitempty"`
	Range *Range            `json:"range,omitempty"`
}

type TimeRangeLabelsMap struct {
	Map   map[string]map[string]string `json:"map,omitempty"`
	Range *Range                       `json:"range,omitempty"`
}

func ToTimeRangeLabels(l *Labels) ([]*TimeRangeLabels, error) {
	if l == nil {
		return nil, fmt.Errorf("nil lables")
	}
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

func Rearrange(mtrl map[string][]*TimeRangeLabels, d *Discovery) []*TimeRangeLabelsMap {
	st := make(map[int64]bool)
	for k, trls := range mtrl {
		n := len(trls)
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
				mtrl[k] = trls
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
	res := make([]*TimeRangeLabelsMap, n)
	for i, ar := range actualRanges {
		trlm := &TimeRangeLabelsMap{Map: make(map[string]map[string]string), Range: ar}
		for k, trls := range mtrl {
			for _, trl := range trls {
				if trl.Range.Contains(ar) {
					trlm.Map[k] = trl.Map
				}
				break
			}
		}
		res[i] = trlm
	}
	return res
}
