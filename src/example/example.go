package main

import (
	errplane "../.."
	"os"
	"time"
)

const (
	appKey      = ""
	apiKey      = ""
	environment = ""
	proxy       = ""
)

func main() {
	ep := errplane.New("w.apiv3.errplane.com", "udp.apiv3.errplane.com", appKey, environment, apiKey)
	if proxy != "" {
		ep.SetProxy(proxy)
	}
	err := ep.Report("some_metric", 123.4, time.Now(), "some_context", errplane.Dimensions{
		"foo": "bar",
	})
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}
