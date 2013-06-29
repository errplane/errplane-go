package errplane

import (
	"fmt"
	. "launchpad.net/gocheck"
	"net"
	"time"
)

type ErrplaneAggregatorApiSuite struct{}

var _ = Suite(&ErrplaneAggregatorApiSuite{})

var (
	udpListener *net.UDPConn
	udpRecorder *UdpRequestRecorder
)

type UdpRequestRecorder struct {
	requests [][]byte
}

func (self *UdpRequestRecorder) recordRequest(conn net.Conn) {
}

func (s *ErrplaneAggregatorApiSuite) SetUpSuite(c *C) {
	var err error
	addr, err := net.ResolveUDPAddr("udp4", "")
	c.Assert(err, IsNil)
	udpListener, err = net.ListenUDP("udp4", addr)
	c.Assert(err, IsNil)
	udpRecorder = new(UdpRequestRecorder)
	go func() {
		for {
			buffer := make([]byte, 1024)
			n, _, _, _, err := udpListener.ReadMsgUDP(buffer, nil)
			if err != nil || n <= 0 {
				break
			}
			udpRecorder.requests = append(udpRecorder.requests, buffer[:n])
		}
	}()
	currentTime = time.Now()
}

func (s *ErrplaneAggregatorApiSuite) TearDownSuite(c *C) {
	udpListener.Close()
}

func (s *ErrplaneAggregatorApiSuite) TestApi(c *C) {
	ep := newTestClient("", udpListener.LocalAddr().(*net.UDPAddr).String(), "app4you2love", "staging", "some_key")
	c.Assert(ep, NotNil)

	err := ep.ReportUDP("some_metric", 123.4, "some_context", Dimensions{
		"foo": "bar",
	})
	c.Assert(err, IsNil)
	time.Sleep(200 * time.Millisecond)
	err = ep.Count("some_metric", 10, "some_context", Dimensions{
		"foo": "bar",
	})
	c.Assert(err, IsNil)
	time.Sleep(200 * time.Millisecond)
	err = ep.Aggregate("some_metric", 234.5, "some_context", Dimensions{
		"foo": "bar",
	})
	c.Assert(err, IsNil)
	time.Sleep(200 * time.Millisecond)
	c.Assert(udpRecorder.requests, HasLen, 3)
	expected := fmt.Sprintf(`{"a":"some_key","d":"app4you2lovestaging","o":"r","w":[{"n":"some_metric","p":[{"c":"some_context","d":{"foo":"bar"},"v":123.4}]}]}`)
	c.Assert(string(udpRecorder.requests[0]), Equals, expected)
	expected = fmt.Sprintf(`{"a":"some_key","d":"app4you2lovestaging","o":"c","w":[{"n":"some_metric","p":[{"c":"some_context","d":{"foo":"bar"},"v":10}]}]}`)
	c.Assert(string(udpRecorder.requests[1]), Equals, expected)
	expected = fmt.Sprintf(`{"a":"some_key","d":"app4you2lovestaging","o":"t","w":[{"n":"some_metric","p":[{"c":"some_context","d":{"foo":"bar"},"v":234.5}]}]}`)
	c.Assert(string(udpRecorder.requests[2]), Equals, expected)
}
