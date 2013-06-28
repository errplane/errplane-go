package errplane

import (
	"fmt"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"net"
	"net/http"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

type ErrplaneApiSuite struct{}

var (
	_           = Suite(&ErrplaneApiSuite{})
	recorder    *HttpRequestRecorder
	listener    net.Listener
	currentTime time.Time
)

type HttpRequestRecorder struct {
	requests [][]byte
}

func (self *HttpRequestRecorder) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	data, _ := ioutil.ReadAll(req.Body)
	self.requests = append(self.requests, data)
}

func (s *ErrplaneApiSuite) SetUpSuite(c *C) {
	var err error
	listener, err = net.Listen("tcp4", "")
	c.Assert(err, IsNil)
	recorder = new(HttpRequestRecorder)
	http.Handle("/databases/app4you2lovestaging/write_keys", recorder)
	go func() { http.Serve(listener, nil) }()

	currentTime = time.Now()
}

func (s *ErrplaneApiSuite) TearDownSuite(c *C) {
	listener.Close()
}

func (s *ErrplaneApiSuite) TestApi(c *C) {
	ep := newTestClient(listener.Addr().(*net.TCPAddr).String(), "app4you2love", "staging", "some_key")
	c.Assert(ep, NotNil)

	ep.Report("some_metric", 123.4, currentTime, "some_context", Dimensions{
		"foo": "bar",
	})
	c.Assert(recorder.requests, HasLen, 1)
	expected := fmt.Sprintf(
		`[{"n":"some_metric","p":[{"c":"some_context","d":{"foo":"bar"},"t":%d,"v":123.4}]}]`,
		currentTime.UnixNano()/int64(time.Millisecond))
	c.Assert(string(recorder.requests[0]), Equals, expected)
}
