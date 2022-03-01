package datamodel

import (
	"encoding/json"
	"fmt"
	"time"
)

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
