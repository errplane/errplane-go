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
	proto    string
	httpHost string
	udpAddr  string
	url      string
	apiKey   string
	database string
	Timeout  time.Duration
}

const (
	DEFAULT_HTTP_HOST = "w.apiv3.errplane.com"
	DEFAULT_UDP_ADDR  = "udp.apiv3.errplane.com:8126"
)

// Initializer.
//   app: the application key from the Settings/Applications page
//   environment: the environment from the Settings/Applications page
//   apiKey: the api key from Settings/Orginzations page
func New(app, environment, apiKey string) *Errplane {
	return newCommon("https", app, environment, apiKey)
}

func newTestClient(app, environment, apiKey string) *Errplane {
	return newCommon("http", app, environment, apiKey)
}

func newCommon(proto, app, environment, apiKey string) *Errplane {
	database := fmt.Sprintf("%s%s", app, environment)
	ep := &Errplane{
		proto:    proto,
		database: database,
		udpAddr:  DEFAULT_UDP_ADDR,
		apiKey:   apiKey,
		Timeout:  1 * time.Second,
	}
	return ep.initUrl()
}

func (self *Errplane) SetHttpHost(host string) {
	self.httpHost = host
	self.initUrl()
}

func (self *Errplane) SetUdpAddr(addr string) {
	self.udpAddr = addr
}

func (self *Errplane) initUrl() *Errplane {
	params := url.Values{}
	params.Set("api_key", self.apiKey)
	self.url = fmt.Sprintf("%s://%s/databases/%s/points?%s", self.proto, self.httpHost, self.database, params.Encode())
	return self
}

func (self *Errplane) SetProxy(proxy string) error {
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		return err
	}
	http.DefaultTransport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
	return nil
}

// FIXME: make timestamp, context and dimensions optional (accept empty values, e.g. nil)
func (self *Errplane) Report(metric string, value float64, timestamp time.Time,
	context string, dimensions Dimensions) error {
	data := Metrics{
		Metric{
			"n": metric,
			"p": Points{
				Point{
					"t": timestamp.Unix(),
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
	remoteAddr, err := net.ResolveUDPAddr("udp4", self.udpAddr)
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

func (self *Errplane) Sum(metric string, value int, context string, dimensions Dimensions) error {
	return self.sendUdpPayload("c", metric, float64(value), context, dimensions)
}
