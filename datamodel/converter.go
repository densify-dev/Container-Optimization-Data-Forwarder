package datamodel

import (
	"fmt"
	"github.com/prometheus/common/model"
	"math"
	"time"
)

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

func timeConv(value float64) fmt.Stringer {
	sec, frac := math.Modf(value)
	return time.Unix(int64(sec), int64(float64(time.Second)*frac))
}

func boolConv(value float64) fmt.Stringer {
	b := value != 0.0
	return &BoolStringer{Value: b}
}
