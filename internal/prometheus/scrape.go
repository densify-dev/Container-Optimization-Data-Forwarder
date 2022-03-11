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

const (
	scrapeIntervalDiscoveredLabelName = "__scrape_interval__"
	scrapeTimeoutDiscoveredLabelName  = "__scrape_timeout__"
)

func conditionallyAddDropped(intervals map[string]*time.Duration, exporter string, dt *v1.DroppedTarget) {
	interval := parseDurationLabel(dt, scrapeIntervalDiscoveredLabelName)
	timeout := parseDurationLabel(dt, scrapeTimeoutDiscoveredLabelName)
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
	if at != nil {
		if mts, f := labelsMatchers[exporter]; f && len(mts) > 0 {
			if matchLabelSet(at.Labels, exporter, mts...) {
				return true
			}
		}
		if mts, f := discoveredLabelsMatchers[exporter]; f && len(mts) > 0 {
			if matchMap(at.DiscoveredLabels, exporter, mts...) {
				return true
			}
		}
		for _, mt := range getFieldMatchers(at, exporter) {
			if mt.match(exporter) {
				return true
			}
		}
	}
	return false
}

func isDroppedExporter(dt *v1.DroppedTarget, exporter string) bool {
	if mts, f := discoveredLabelsMatchers[exporter]; f && len(mts) > 0 {
		if matchMap(dt.DiscoveredLabels, exporter, mts...) {
			return true
		}
	}
	return false
}

func matchMap(m map[string]string, s string, mts ...*matcher) bool {
	for _, mt := range mts {
		// replace the key - which is s in mt - by the value from m
		if v, ok := m[mt.s]; ok {
			mt1 := &matcher{f: mt.f, s: v}
			if mt1.match(s) {
				return true
			}
		}
	}
	return false
}

func matchLabelSet(ls model.LabelSet, s string, mts ...*matcher) bool {
	m := make(map[string]string, len(ls))
	for k, v := range ls {
		m[string(k)] = string(v)
	}
	return matchMap(m, s, mts...)
}

type matchFunc func(string, string) bool

type matcher struct {
	f matchFunc
	s string
}

func (m *matcher) match(s string) bool {
	return m.f(m.s, s)
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

var labelsMatchers = map[string][]*matcher{
	ne: {
		{f: exact, s: "service"},
		{f: contains, s: "job"},
		{f: contains, s: "pod"},
		{f: exact, s: "component"},
		{f: contains, s: "kubernetes_name"},
		{f: contains, s: "app"},
		{f: contains, s: "name"},
	},
	cad: {
		{f: contains, s: "job"},
		{f: contains, s: "metrics_path"},
	},
	ksm: {
		{f: contains, s: "service"},
		{f: contains, s: "pod"},
		{f: contains, s: "job"},
		{f: contains, s: "app_kubernetes_io_name"},
		{f: contains, s: "kubernetes_name"},
	},
	ossm: {},
}

var discoveredLabelsMatchers = map[string][]*matcher{
	ne: {
		{f: contains, s: "__meta_kubernetes_endpoints_name"},
		{f: contains, s: "__meta_kubernetes_pod_container_name"},
		{f: contains, s: "__meta_kubernetes_pod_controller_name"},
		{f: contains, s: "__meta_kubernetes_pod_label_app"},
		{f: contains, s: "__meta_kubernetes_pod_name"},
		{f: exact, s: "__meta_kubernetes_pod_label_component"},
		{f: contains, s: "__meta_kubernetes_service_name"},
		{f: contains, s: "job"},
	},
	cad: {
		{f: contains, s: "__metrics_path__"},
		{f: contains, s: "job"},
	},
	ksm: {
		{f: contains, s: "__meta_kubernetes_endpoints_name"},
		{f: contains, s: "__meta_kubernetes_pod_container_name"},
		{f: contains, s: "__meta_kubernetes_pod_controller_name"},
		{f: contains, s: "__meta_kubernetes_pod_label_app"},
		{f: contains, s: "__meta_kubernetes_pod_name"},
		{f: contains, s: "__meta_kubernetes_service_name"},
		{f: contains, s: "job"},
	},
	ossm: nil,
}

func getFieldMatchers(at *v1.ActiveTarget, exporter string) []*matcher {
	mts := []*matcher{{f: contains, s: at.ScrapePool}}
	switch exporter {
	case cad:
		mts = append(mts, &matcher{f: contains, s: at.ScrapeURL},
			&matcher{f: contains, s: at.GlobalURL})
	}
	return mts
}
