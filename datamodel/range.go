package datamodel

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	empty                 = "<nil>"
	delimiter             = "-"
	validFormat           = "nanotimestamp/" + empty
	PrometheusGranularity = time.Millisecond
)

// Range represents a time range. We don't use Prometheus client's Range as
// we want to make Start, End optional (nillable) and make them have json tags,
// and we don't need the Step
type Range struct {
	// The boundaries of the time range.
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// In - Range is inclusive, both ends
func (r *Range) In(t time.Time) bool {
	return (r.Start == nil || r.Start.Before(t) || r.Start.Equal(t)) &&
		(r.End == nil || r.End.After(t) || r.End.Equal(t))
}

// Equal - return true if r is exactly equal to other
func (r *Range) Equal(other *Range) bool {
	if other == nil {
		return false
	} else {
		return equal(r.Start, other.Start) && equal(r.End, other.End)
	}
}

// Before - applies only to the Start field, for consistency use only for non-overlapping ranges
func (r *Range) Before(other *Range) bool {
	if other == nil {
		return false
	} else {
		return before(r.Start, other.Start)
	}
}

// After - applies only to the Start field, for consistency use only for non-overlapping ranges
func (r *Range) After(other *Range) bool {
	if other == nil {
		return false
	} else {
		return after(r.Start, other.Start)
	}
}

// IsDistinct - returns if there is NO overlap between the ranges
func (r *Range) IsDistinct(other *Range) bool {
	if other == nil {
		return false
	} else {
		return before(r.End, other.Start) || before(other.End, r.Start)
	}
}

// Overlaps - returns if there is ANY overlap between the ranges
func (r *Range) Overlaps(other *Range) bool {
	return !r.IsDistinct(other)
}

// Intersection - returns if there is ANY overlap between the ranges
func (r *Range) Intersection(other *Range) *Range {
	var res *Range
	if other == nil {
		res = r
	} else if r.Overlaps(other) {
		var start, end *time.Time
		if otherStart := r.Start == nil || (other.Start != nil && r.Start.Before(*other.Start)); otherStart {
			start = other.Start
		} else {
			start = r.Start
		}
		if otherEnd := r.End == nil || (other.End != nil && r.End.After(*other.End)); otherEnd {
			end = other.End
		} else {
			end = r.End
		}
		res = &Range{Start: start, End: end}
	}
	return res
}

// Contains - returns if r contains other
func (r *Range) Contains(other *Range) bool {
	if other == nil {
		return false
	} else {
		return (before(r.Start, other.Start) || equal(r.Start, other.Start)) &&
			(before(other.End, r.End) || equal(other.End, r.End))
	}
}

func (r *Range) AdjustRange(other *Range, dev *time.Duration, adjustStart, adjustEnd bool) {
	if other == nil || dev == nil || *dev <= 0 || !(adjustStart || adjustEnd) {
		return
	}
	nns := other.Start != nil
	nne := other.End != nil
	if nns && nne && other.End.Sub(*other.Start) < *dev {
		return
	}
	if adjustStart {
		if nns && r.Start != nil {
			rng := &Range{Start: other.Start, End: add(other.Start, *dev)}
			if rng.In(*r.Start) {
				r.Start = other.Start
			}
		}
	}
	if adjustEnd {
		if nne && r.End != nil {
			rng := &Range{Start: add(other.End, -*dev), End: other.End}
			if rng.In(*r.End) {
				r.End = other.End
			}
		}
	}
}

func (r *Range) StretchTo(other *Range) {
	r.stretch(other, self)
}

func (r *Range) StretchOther(other *Range) {
	r.stretch(other, another)
}

func (r *Range) StretchBoth(other *Range) {
	r.stretch(other, both)
}

type stretchType int

const (
	self stretchType = iota
	another
	both
)

func (r *Range) stretch(other *Range, arg stretchType) {
	if other == nil || other.Start == nil || r.End == nil || !other.Start.After((*r.End).Add(PrometheusGranularity)) {
		return
	}
	var rEnd, otherStart *time.Time
	switch arg {
	case self:
		rEnd = add(other.Start, -PrometheusGranularity)
	case another:
		otherStart = add(r.End, PrometheusGranularity)
	case both:
		if d := (other.Start.Sub(*r.End)) / 2; d >= PrometheusGranularity {
			rEnd = add(r.End, d)
			otherStart = add(rEnd, PrometheusGranularity)
		}
	}
	if rEnd != nil {
		r.End = rEnd
	}
	if otherStart != nil {
		other.Start = otherStart
	}
}

func add(t *time.Time, d time.Duration) *time.Time {
	var res *time.Time
	if t != nil {
		rt := t.Add(d)
		res = &rt
	}
	return res
}

// String - implement fmt.Stringer
func (r *Range) String() string {
	return format(str(r.Start), str(r.End))
}

// Parse string s to a Range
func Parse(s string) (*Range, error) {
	c := strings.Split(s, delimiter)
	if len(c) == 2 {
		var start, end *time.Time
		var err error
		if start, err = parse(c[0]); err != nil {
			return nil, err
		}
		if end, err = parse(c[1]); err != nil {
			return nil, err
		}
		return &Range{Start: start, End: end}, nil
	} else {
		return nil, fmt.Errorf("%s fails to meet format %s", s, format(validFormat, validFormat))
	}
}

// Expand expands a Range with a *time.Time t according to the following logic:
// * if t is nil, do nothing
// * if both Start and End are nil, set both to t
// * if Start is nil and End isn't, then if End is before t, set End to t
// * if End is nil and Start isn't, then if Start is after t, set Start to t
// * if both Start and End are non-nil, then:
// * * if Start is after t, set Start to t; else if End is before t, set End to t
func (r *Range) Expand(t *time.Time) {
	var setStart, setEnd bool
	if t != nil {
		if r.Start == nil {
			if r.End == nil {
				setStart = true
				setEnd = true
			} else {
				setEnd = r.End.Before(*t)
			}
		} else {
			setStart = r.Start.After(*t)
			if !setStart && r.End != nil {
				setEnd = r.End.Before(*t)
			}
		}
	}
	if setStart {
		r.Start = t
	}
	if setEnd {
		r.End = t
	}
}

// ExpandRange expands the range with Start and End of provided argument
func (r *Range) ExpandRange(other *Range) {
	if other != nil {
		r.Expand(other.Start)
		r.Expand(other.End)
	}
}

func equal(t1, t2 *time.Time) bool {
	if t1 == nil {
		return t2 == nil
	} else {
		return t2 != nil && t1.Equal(*t2)
	}
}

func before(t1, t2 *time.Time) bool {
	if t1 == nil {
		return t2 != nil
	} else {
		return t2 != nil && t1.Before(*t2)
	}
}

func after(t1, t2 *time.Time) bool {
	if t1 == nil {
		return t2 != nil
	} else {
		return t2 != nil && t1.After(*t2)
	}
}

func str(t *time.Time) string {
	if t == nil {
		return empty
	} else {
		return fmt.Sprintf("%d", t.UnixNano())
	}
}

func parse(s string) (*time.Time, error) {
	if s == empty {
		return nil, nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		t := time.Unix(0, n)
		return &t, nil
	} else {
		return nil, err
	}
}

func format(s1, s2 string) string {
	return strings.Join([]string{s1, s2}, delimiter)
}
