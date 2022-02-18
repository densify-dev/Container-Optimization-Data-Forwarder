package datamodel

import (
	"time"
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
