package prometheus

// patch until prometheus go-client supports scrapeInterval, scrapeTimeout in Targets()

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"net/http"
	"time"
)

const (
	epTargets = "/api/v1/targets"
)

type promAPIv2 struct {
	v1.API
	c api.Client
}

func (pav2 *promAPIv2) TargetsV2(ctx context.Context) (TargetsResult, error) {
	u := pav2.c.URL(epTargets, nil)

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return TargetsResult{}, err
	}

	_, body, err := pav2.c.Do(ctx, req)
	if err != nil {
		return TargetsResult{}, err
	}

	var res TargetsResultResponse
	if err = json.Unmarshal(body, &res); err != nil {
		return TargetsResult{}, err
	}
	return *res.Data, nil
}

// TargetsResult contains the result from querying the targets endpoint.
type TargetsResult struct {
	Active  []ActiveTarget     `json:"activeTargets"`
	Dropped []v1.DroppedTarget `json:"droppedTargets"`
}

type TargetsResultResponse struct {
	Status string         `json:"status"`
	Data   *TargetsResult `json:"data"`
}

// ActiveTarget models an active Prometheus scrape target.
type ActiveTarget struct {
	v1.ActiveTarget
	ScrapeInterval *Duration `json:"scrapeInterval,omitempty"`
	ScrapeTimeout  *Duration `json:"scrapeTimeout,omitempty"`
}

type Duration struct {
	Duration time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	if b[0] == '"' {
		sd := string(b[1 : len(b)-1])
		d.Duration, err = time.ParseDuration(sd)
		return
	}
	var id int64
	id, err = json.Number(b).Int64()
	d.Duration = time.Duration(id)
	return
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.Duration.String())), nil
}
