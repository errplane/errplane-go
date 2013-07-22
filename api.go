package errplane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Point map[string]interface{}
type Points []Point
type Metric map[string]interface{}
type Metrics []Metric
type UdpPayload map[string]interface{}
type Dimensions map[string]string

type Errplane struct {
	url      string
	apiKey   string
	database string
	udpHost  string
	Timeout  time.Duration
}

// Initializer.
//   httpHost: the hostname of the collector (e.g. api.errplane.go)
//   udpHost: the hostname of the aggregator (e.g. udp.errplane.go)
//   app: the application key from the Settings/Applications page
//   environment: the environment from the Settings/Applications page
//   apiKey: the api key from Settings/Orginzations page
func New(httpHost, udpHost, app, environment, apiKey string) *Errplane {
	return newCommon("https", httpHost, udpHost, app, environment, apiKey)
}

func newTestClient(httpHost, udpHost, app, environment, apiKey string) *Errplane {
	return newCommon("http", httpHost, udpHost, app, environment, apiKey)
}

func newCommon(proto, httpHost, udpHost, app, environment, apiKey string) *Errplane {
	database := fmt.Sprintf("%s%s", app, environment)
	params := url.Values{}
	params.Set("api_key", apiKey)
	return &Errplane{
		database: database,
		url:      fmt.Sprintf("%s://%s/databases/%s/points?%s", proto, httpHost, database, params.Encode()),
		udpHost:  udpHost,
		apiKey:   apiKey,
		Timeout:  1 * time.Second,
	}
}

func (self *Errplane) Report(metric string, value float64, timestamp time.Time,
	context string, dimensions Dimensions) error {
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
	if resp.StatusCode != 201 {
		return fmt.Errorf("Server returned status code %d", resp.StatusCode)
	}
	return nil
}

func (self *Errplane) sendUdpPayload(metricType, metric string, value float64, context string, dimensions Dimensions) error {
	localAddr, err := net.ResolveUDPAddr("udp4", "")
	if err != nil {
		return err
	}
	remoteAddr, err := net.ResolveUDPAddr("udp4", self.udpHost)
	if err != nil {
		return err
	}
	udpConn, err := net.DialUDP("udp4", localAddr, remoteAddr)
	if err != nil {
		return err
	}
	data := Metric{
		"d": self.database,
		"a": self.apiKey,
		"o": metricType,
		"w": Metrics{
			Metric{
				"n": metric,
				"p": Points{
					Point{
						"v": value,
						"c": context,
						"d": dimensions,
					},
				},
			},
		},
	}
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = udpConn.Write(buf)
	return err
}

func (self *Errplane) ReportUDP(metric string, value float64, context string, dimensions Dimensions) error {
	return self.sendUdpPayload("r", metric, value, context, dimensions)
}

func (self *Errplane) Aggregate(metric string, value float64, context string, dimensions Dimensions) error {
	return self.sendUdpPayload("t", metric, value, context, dimensions)
}

func (self *Errplane) Count(metric string, value int, context string, dimensions Dimensions) error {
	return self.sendUdpPayload("c", metric, float64(value), context, dimensions)
}
