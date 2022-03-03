package datamodel

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/r3labs/diff/v2"
	"sort"
	"time"
)

type RangeLabels struct {
	Map   map[string]string `json:"map,omitempty"`
	Range *Range            `json:"range,omitempty"`
}

type RangeLabelsMap struct {
	Map   map[string]map[string]map[string]string `json:"map,omitempty"`
	Range *Range                                  `json:"range,omitempty"`
}

type MapOfRangeLabels map[string]map[string][]*RangeLabels

func ToRangeLabels(l *Labels) ([]*RangeLabels, error) {
	if l == nil {
		return nil, fmt.Errorf("nil lables")
	}
	var res []*RangeLabels
	var err error
	if l.Original != nil {
		res = append(res, l.Original)
		l.setCurrents(l.Original.Map, l.Original.Range)
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
					res = append(res, &RangeLabels{Map: m, Range: d.Range})
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

func AppendRangeLabels(mrl MapOfRangeLabels, l *Labels, k string) (MapOfRangeLabels, error) {
	lm := LabelMap{k: l}
	return AppendRangeLabelMap(mrl, lm, Unmapped)
}

func AppendRangeLabelMap(mrl MapOfRangeLabels, lm LabelMap, lmk string) (MapOfRangeLabels, error) {
	var err error
	res := mrl
	if res == nil {
		res = make(MapOfRangeLabels)
	}
	m, f := res[lmk]
	if !f {
		m = make(map[string][]*RangeLabels)
		res[lmk] = m
	}
	for k, l := range lm {
		if rl, e := ToRangeLabels(l); e == nil {
			m[k] = rl
		} else {
			err = multierror.Append(err, e)
		}
	}
	return res, err
}

func Rearrange(mrl MapOfRangeLabels, d *Discovery) []*RangeLabelsMap {
	st := make(map[int64]bool)
	for k1, m := range mrl {
		for k2, rls := range m {
			n := len(rls)
			for i, rl := range rls {
				r := rl.Range
				r.AdjustRange(d.Range, d.MaxScrapeInterval, i == 0, i == n-1)
				// now if rl is not the last one, adjust its end and the start of the next one to the
				// median of the gap between them
				if i < n-1 {
					var nextRange *Range
					if next := rls[i+1]; next != nil {
						nextRange = next.Range
					}
					r.StretchBoth(nextRange)
				}
				// now record the stat time of the range
				st[r.Start.UnixNano()] = true
			}
			// if we are missing ranges to cover the entire discovery range, add additional RangeLabels
			// with an empty map at the beginning/end of the slice
			if n > 0 {
				var modified bool
				start := rls[0].Range.Start
				if !start.Equal(*d.Range.Start) {
					rng := &Range{Start: d.Range.Start, End: add(start, -time.Nanosecond)}
					newStart := &RangeLabels{Range: rng}
					rls = append([]*RangeLabels{newStart}, rls...)
					st[rng.Start.UnixNano()] = true
					modified = true
				}
				end := rls[n-1].Range.End
				if !end.Equal(*d.Range.End) {
					rng := &Range{Start: add(end, time.Nanosecond), End: d.Range.End}
					newEnd := &RangeLabels{Range: rng}
					rls = append(rls, newEnd)
					st[rng.Start.UnixNano()] = true
					modified = true
				}
				if modified {
					mrl[k1][k2] = rls
				}
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
	res := make([]*RangeLabelsMap, n)
	for i, ar := range actualRanges {
		rlm := &RangeLabelsMap{Map: make(map[string]map[string]map[string]string), Range: ar}
		for k1, m := range mrl {
			rlm.Map[k1] = make(map[string]map[string]string)
			for k2, rls := range m {
				for _, rl := range rls {
					if rl.Range.Contains(ar) {
						rlm.Map[k1][k2] = rl.Map
					}
					break
				}
			}
			res[i] = rlm
		}
	}
	return res
}
