package main

import (
	errplane "../.."
	"math/rand"
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
	ep := errplane.New("w.apiv3.errplane.com", "udp.apiv3.errplane.com:8126", appKey, environment, apiKey)
	if proxy != "" {
		ep.SetProxy(proxy)
	}
	err := ep.Report("some_metric", 123.4, time.Now(), "some_context", errplane.Dimensions{
		"foo": "bar",
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		value := rand.Float64() * 100
		err = ep.Aggregate("some_aggregate", value, "", nil)
		if err != nil {
			panic(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	os.Exit(0)
}
