package datamodel

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	empty       = "<nil>"
	delimiter   = "-"
	validFormat = "nanotimestamp/" + empty
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
