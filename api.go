package errplane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Point map[string]interface{}
type Points []Point
type Metric map[string]interface{}
type Metrics []Metric
type Dimensions map[string]string

type Errplane struct {
	url     string
	apiKey  string
	Timeout time.Duration
}

func New(host, app, environment, apiKey string) *Errplane {
	return newCommon("https", host, app, environment, apiKey)
}

func newTestClient(host, app, environment, apiKey string) *Errplane {
	return newCommon("http", host, app, environment, apiKey)
}

func newCommon(proto, host, app, environment, apiKey string) *Errplane {
	return &Errplane{
		url:     fmt.Sprintf("%s://%s/databases/%s%s/write_keys", proto, host, app, environment),
		apiKey:  apiKey,
		Timeout: 1 * time.Second,
	}
}

func (self *Errplane) Report(metric string, value float64, timestamp time.Time, context string, dimensions Dimensions) error {
	data := Metrics{
		Metric{
			"n": metric,
			"p": Points{
				Point{
					"t": timestamp.UnixNano() / int64(time.Millisecond),
					"v": value,
					"c": context,
					"d": dimensions,
				},
			},
		},
	}
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	resp, err := http.Post(self.url, "application/json", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Server returned status code %d", resp.StatusCode)
	}
	return nil
}
