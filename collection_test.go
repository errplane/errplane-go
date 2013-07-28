package errplane

import (
	"fmt"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"net"
	"net/http"
	"net/url"
	"time"
)

type ErrplaneCollectorApiSuite struct{}

var _ = Suite(&ErrplaneCollectorApiSuite{})

var (
	recorder    *HttpRequestRecorder
	listener    net.Listener
	currentTime time.Time
)

type HttpRequestRecorder struct {
	requests [][]byte
	forms    []url.Values
}

func (self *HttpRequestRecorder) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	data, _ := ioutil.ReadAll(req.Body)
	self.requests = append(self.requests, data)
	req.ParseForm()
	self.forms = append(self.forms, req.Form)
}

func (s *ErrplaneCollectorApiSuite) SetUpSuite(c *C) {
	var err error
	listener, err = net.Listen("tcp4", "")
	c.Assert(err, IsNil)
	recorder = new(HttpRequestRecorder)
	http.Handle("/databases/app4you2lovestaging/points", recorder)
	go func() { http.Serve(listener, nil) }()

	currentTime = time.Now()
}

func (s *ErrplaneCollectorApiSuite) SetUpTest(c *C) {
	recorder.requests = nil
	recorder.forms = nil
}

func (s *ErrplaneCollectorApiSuite) TearDownSuite(c *C) {
	listener.Close()
}

func (s *ErrplaneCollectorApiSuite) TestApi(c *C) {
	ep := newTestClient("app4you2love", "staging", "some_key")
	c.Assert(ep, NotNil)
	ep.SetHttpHost(listener.Addr().(*net.TCPAddr).String())

	ep.Report("some_metric", 123.4, currentTime, "some_context", Dimensions{
		"foo": "bar",
	})

	ep.Close() // make sure we flush all the points

	c.Assert(recorder.requests, HasLen, 1)
	expected := fmt.Sprintf(
		`[{"n":"some_metric","p":[{"c":"some_context","d":{"foo":"bar"},"t":%d,"v":123.4}]}]`,
		currentTime.UnixNano()/int64(time.Second))
	c.Assert(string(recorder.requests[0]), Equals, expected)
	c.Assert(recorder.forms, HasLen, 1)
	c.Assert(recorder.forms[0].Get("api_key"), Equals, "some_key")
}

func (s *ErrplaneCollectorApiSuite) TestApiAggregatesPoints(c *C) {
	ep := newTestClient("app4you2love", "staging", "some_key")
	c.Assert(ep, NotNil)
	ep.SetHttpHost(listener.Addr().(*net.TCPAddr).String())

	ep.Report("some_metric", 123.4, currentTime, "some_context", Dimensions{
		"foo": "bar",
	})

	ep.Report("some_metric", 567.8, currentTime, "different_context", Dimensions{
		"foo": "bar",
	})

	ep.Report("different_metric", 123.4, currentTime, "some_context", Dimensions{
		"foo": "bar",
	})

	ep.Close() // make sure we flush all the points

	c.Assert(recorder.requests, HasLen, 1)
	epocTime := currentTime.UnixNano() / int64(time.Second)
	expected := fmt.Sprintf(
		`[{"n":"some_metric","p":[{"c":"some_context","d":{"foo":"bar"},"t":%d,"v":123.4},{"c":"different_context","d":{"foo":"bar"},"t":%d,"v":567.8}]},{"n":"different_metric","p":[{"c":"some_context","d":{"foo":"bar"},"t":%d,"v":123.4}]}]`, epocTime, epocTime, epocTime)
	c.Assert(string(recorder.requests[0]), Equals, expected)
	c.Assert(recorder.forms, HasLen, 1)
	c.Assert(recorder.forms[0].Get("api_key"), Equals, "some_key")
}
