package datamodel

import (
	"fmt"
	"github.com/prometheus/common/model"
	"math"
	"time"
)

const (
	TimeFormat = time.RFC3339Nano
)

type ValueConversionFunction func(float64) float64
type StringerConversionFunction func(float64) fmt.Stringer
type FilterFunc func(v float64) bool

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

// TimeStringer is required to format the time using RFC3339Nano (same as MarshalJSON)
// i.s.o. using time.Time.String()
type TimeStringer struct {
	Value *time.Time
}

func (ts *TimeStringer) String() string {
	var s string
	if ts.Value != nil {
		s = ts.Value.Format(TimeFormat)
	}
	return s
}

func ToBool(value float64) bool {
	return value != 0.0
}

func timeConv(value float64) fmt.Stringer {
	sec, frac := math.Modf(value)
	t := time.Unix(int64(sec), int64(float64(time.Second)*frac))
	return &TimeStringer{Value: &t}
}

func boolConv(value float64) fmt.Stringer {
	return &BoolStringer{Value: ToBool(value)}
}
