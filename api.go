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

type PostType int

const (
	UDP PostType = iota
	HTTP
)

type ErrplanePost struct {
	postType PostType
	data     interface{}
}

type Errplane struct {
	proto     string
	httpHost  string
	udpAddr   string
	url       string
	apiKey    string
	database  string
	Timeout   time.Duration
	closeChan chan bool
	msgChan   chan *ErrplanePost
	closed    bool
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
		httpHost:  DEFAULT_HTTP_HOST,
		udpAddr:   DEFAULT_UDP_ADDR,
		proto:     proto,
		database:  database,
		apiKey:    apiKey,
		Timeout:   1 * time.Second,
		msgChan:   make(chan *ErrplanePost),
		closeChan: make(chan bool),
		closed:    false,
	}
	ep.initUrl()
	go ep.processMessages()
	return ep
}

// call from a goroutine, this method never returns
func (self *Errplane) processMessages() {
	posts := make([]*ErrplanePost, 0)
	for {

		select {
		case x := <-self.msgChan:
			posts = append(posts, x)
			if len(posts) < 100 {
				continue
			}
			self.flushPosts(posts)
		case <-time.After(1 * time.Second):
			self.flushPosts(posts)
		case <-self.closeChan:
			self.flushPosts(posts)
			self.closeChan <- true
			return
		}

		posts = make([]*ErrplanePost, 0)
	}
}

func (self *Errplane) flushPosts(posts []*ErrplanePost) error {
	if len(posts) == 0 {
		return nil
	}

	httpPoints := make([]Metrics, 0)
	udpReportPoints := make([]Metrics, 0)
	udpSumPoints := make([]Metrics, 0)
	udpAggregatePoints := make([]Metrics, 0)

	for _, post := range posts {
		if post.postType == UDP {
			metric := post.data.(Metric)
			switch metric["o"] {
			case "r":
				udpReportPoints = append(udpReportPoints, metric["w"].(Metrics))
			case "t":
				udpAggregatePoints = append(udpAggregatePoints, metric["w"].(Metrics))
			case "c":
				udpSumPoints = append(udpSumPoints, metric["w"].(Metrics))
			}
		} else {
			httpPoints = append(httpPoints, post.data.(Metrics))
		}
	}

	// do the http ones first
	httpPoint := mergeMetrics(httpPoints)
	self.sendHttp(httpPoint)

	// do the udp points here
	udpReportPoint := mergeMetrics(udpReportPoints)
	if len(udpReportPoints) > 0 {
		self.sendUdp("r", udpReportPoint)
	}
	udpAggregatePoint := mergeMetrics(udpAggregatePoints)
	if len(udpAggregatePoints) > 0 {
		self.sendUdp("t", udpAggregatePoint)
	}
	udpSumPoint := mergeMetrics(udpSumPoints)
	if len(udpSumPoints) > 0 {
		self.sendUdp("c", udpSumPoint)
	}

	return nil
}

func (self *Errplane) heartbeat(name string, interval time.Duration) {
	go func() {
		for {
			if self.closed {
				return
			}

			self.Report(name, 1.0, time.Now(), "", nil)
			time.Sleep(interval)
		}
	}()
}

func (self *Errplane) sendHttp(data Metrics) error {
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

func (self *Errplane) sendUdp(metricType string, points Metrics) error {
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
		"w": points,
	}
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = udpConn.Write(buf)
	return err
}

func mergeMetrics(points []Metrics) Metrics {
	metricToPoints := make(map[string]Points)

	for _, metrics := range points {
		for _, point := range metrics {
			metric := point["n"].(string)
			points := point["p"].(Points)

			points = append(metricToPoints[metric], points...)
			metricToPoints[metric] = points
		}
	}

	mergedMetrics := make(Metrics, 0)

	for metric, points := range metricToPoints {
		mergedMetrics = append(mergedMetrics, Metric{
			"n": metric,
			"p": points,
		})
	}

	return mergedMetrics
}

// Close the errplane object and flush all buffered data points
func (self *Errplane) Close() {
	// tell the go routine to finish
	self.closeChan <- true
	// wait for the go routine to finish
	<-self.closeChan
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
	self.msgChan <- &ErrplanePost{HTTP, data}
	return nil
}

func (self *Errplane) sendUdpPayload(metricType, metric string, value float64, context string, dimensions Dimensions) error {
	data := Metric{
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
	self.msgChan <- &ErrplanePost{UDP, data}
	return nil
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
