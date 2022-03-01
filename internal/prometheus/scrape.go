package prometheus

import (
	"context"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"strings"
	"time"
)

func GetScrapeIntervals(args *Parameters) (intervals map[string]*time.Duration, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err = ensureApi(args); err != nil {
		return
	}
	var tr TargetsResult
	if tr, err = promApi.TargetsV2(ctx); err != nil {
		return
	}
	intervals = make(map[string]*time.Duration, len(exporters))
	for _, exporter := range exporters {
		for _, at := range tr.Active {
			if isActiveExporter(&at.ActiveTarget, exporter) {
				conditionallyAddActive(intervals, exporter, &at)
			}
		}
		for _, dt := range tr.Dropped {
			if isDroppedExporter(&dt, exporter) {
				conditionallyAddDropped(intervals, exporter, &dt)
			}
		}
	}
	if len(intervals) == len(exporters) {
		return
	}
	var cfg v1.ConfigResult
	if cfg, err = promApi.Config(ctx); err != nil {
		return
	}
	getIntervalsFromConfig(intervals, cfg)

	return
}

func getIntervalsFromConfig(intervals map[string]*time.Duration, cfg v1.ConfigResult) {

}

func conditionallyAddActive(intervals map[string]*time.Duration, exporter string, at *ActiveTarget) {
	var interval, timeout time.Duration
	if at.ScrapeInterval != nil {
		interval = at.ScrapeInterval.Duration
	}
	if at.ScrapeTimeout != nil {
		timeout = at.ScrapeTimeout.Duration
	}
	addIfExceeds(intervals, exporter, interval, timeout)
}

func conditionallyAddDropped(intervals map[string]*time.Duration, exporter string, dt *v1.DroppedTarget) {
	interval := parseDurationLabel(dt, "__scrape_interval__")
	timeout := parseDurationLabel(dt, "__scrape_timeout__")
	addIfExceeds(intervals, exporter, interval, timeout)
}

func parseDurationLabel(dt *v1.DroppedTarget, labelName string) time.Duration {
	var d time.Duration
	if si, f := dt.DiscoveredLabels[labelName]; f {
		if pd, err := time.ParseDuration(si); err == nil {
			d = pd
		}
	}
	return d
}

func addIfExceeds(intervals map[string]*time.Duration, exporter string, interval, timeout time.Duration) {
	d := interval + timeout
	if d > 0 {
		curr, f := intervals[exporter]
		if set := !f || d > *curr; set {
			intervals[exporter] = &d
		}
	}
}

func isActiveExporter(at *v1.ActiveTarget, exporter string) bool {
	mtss := make([][]*matcher, 3)
	if at != nil {
		switch exporter {
		case ne:
			mtss[0] = []*matcher{{f: exact, s1: "service", s2: ne}, {f: contains, s1: "job", s2: ne}, {f: contains, s1: "pod", s2: ne}}
			mtss[2] = []*matcher{{f: contains, s1: at.ScrapePool, s2: ne}}
		case cad:
			mtss[0] = []*matcher{{f: contains, s1: "job", s2: cad}, {f: contains, s1: "metrics_path", s2: cad}}
			mtss[2] = []*matcher{{f: contains, s1: at.ScrapePool, s2: cad}, {f: contains, s1: at.ScrapeURL, s2: cad}}
		case ksm:
			mtss[0] = []*matcher{{f: contains, s1: "service", s2: ksm}, {f: contains, s1: "job", s2: ksm},
				{f: contains, s1: "app_kubernetes_io_name", s2: ksm}, {f: contains, s1: "kubernetes_name", s2: ksm}}

		case ossm:
		}
		if dlm, f := discoveredLabelsMatchers[exporter]; f {
			mtss[1] = dlm
		}
	}
	return matchActive(mtss, at)
}

func isDroppedExporter(dt *v1.DroppedTarget, exporter string) bool {
	if mts, f := discoveredLabelsMatchers[exporter]; f {
		return matchDropped(mts, dt)
	}
	return false
}

func matchActive(mtss [][]*matcher, at *v1.ActiveTarget) bool {
	if len(mtss) == 3 {
		if mts := mtss[0]; len(mts) > 0 {
			if matchLabelSet(at.Labels, mts...) {
				return true
			}
		}
		if mts := mtss[1]; len(mts) > 0 {
			if matchMap(at.DiscoveredLabels, mts...) {
				return true
			}
		}
		if mts := mtss[2]; len(mts) > 0 {
			for _, mt := range mts {
				if mt.match() {
					return true
				}
			}
		}
	}
	return false
}

func matchDropped(mts []*matcher, dt *v1.DroppedTarget) bool {
	if len(mts) > 0 {
		if matchMap(dt.DiscoveredLabels, mts...) {
			return true
		}
	}
	return false
}

func matchMap(m map[string]string, mts ...*matcher) bool {
	for _, mt := range mts {
		// replace the key - which is s1 in my - by the value from m
		if v, ok := m[mt.s1]; ok {
			mt.s1 = v
			if mt.match() {
				return true
			}
		}
	}
	return false
}

func matchLabelSet(ls model.LabelSet, mts ...*matcher) bool {
	m := make(map[string]string, len(ls))
	for k, v := range ls {
		m[string(k)] = string(v)
	}
	return matchMap(m, mts...)
}

type matchFunc func(string, string) bool

type matcher struct {
	f      matchFunc
	s1, s2 string
}

func (m *matcher) match() bool {
	return m.f(m.s1, m.s2)
}

func exact(s1, s2 string) bool {
	return s1 == s2
}

func contains(s1, s2 string) bool {
	return strings.Contains(s1, s2)
}

func startsWith(s1, s2 string) bool {
	return strings.HasPrefix(s1, s2)
}

func endsWith(s1, s2 string) bool {
	return strings.HasSuffix(s1, s2)
}

var discoveredLabelsMatchers = map[string][]*matcher{
	ne:   {{f: exact, s1: "__meta_kubernetes_pod_label_app", s2: ne}, {f: exact, s1: "__meta_kubernetes_service_label_k8s_app", s2: ne}},
	cad:  nil,
	ksm:  nil,
	ossm: nil,
}
