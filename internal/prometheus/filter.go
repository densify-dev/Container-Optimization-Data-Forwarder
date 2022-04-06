package prometheus

import (
	"github.com/densify-dev/Container-Optimization-Data-Forwarder/datamodel"
	"github.com/prometheus/common/model"
)

// Filter - PromQL is very limited when it comes to filtering data based on values.
// For example, the query `kube_node_status_condition == 1` works, but if you try to apply
// the binary operator for a range - `kube_node_status_condition[1h] == 1` or
// `(kube_node_status_condition == 1)[1h]` PromQL returns a vague syntax error or
// "binary expression must contain only scalar and instant vector types".
// One can run the query `(kube_node_status_condition == 1)[1h:1s]`, but that one doesn't
// return the raw datapoints - it rather duplicates the value for each second (this 1s
// "resolution", in PromQL terms, has to be smaller than the scrape interval.
// So we opt to getting the raw datapoints, and filtering the values ourselves.
// See also discussion at
// https://stackoverflow.com/questions/46697754/filter-prometheus-results-by-metric-value-not-by-label-value
func Filter(mat model.Matrix, ff datamodel.FilterFunc) model.Matrix {
	if ff == nil {
		return mat
	}
	var newMat model.Matrix
	for _, stream := range mat {
		var newStream *model.SampleStream
		for _, sp := range stream.Values {
			if ff(float64(sp.Value)) {
				if newStream == nil {
					newStream = &model.SampleStream{Metric: stream.Metric}
				}
				newStream.Values = append(newStream.Values, sp)
			}
		}
		if newStream != nil {
			newMat = append(newMat, newStream)
		}
	}
	return newMat
}

func FilterTrue(mat model.Matrix) model.Matrix {
	return Filter(mat, datamodel.ToBool)
}

func FilterPositive(mat model.Matrix) model.Matrix {
	return Filter(mat, positive)
}

func FilterNonNegative(mat model.Matrix) model.Matrix {
	return Filter(mat, nonNegative)
}

func positive(v float64) bool {
	return v > 0.0
}

func nonNegative(v float64) bool {
	return v >= 0.0
}
