package main

import (
	errplane "../.."
	"math/rand"
	"os"
	"time"
)

const (
	appKey      = "app4you2love"
	apiKey      = "962cdc9b-15e7-4b25-9a0d-24a45cfc6bc1"
	environment = "staging"
	proxy       = ""
)

func main() {
	ep := errplane.New(appKey, environment, apiKey)
	if proxy != "" {
		ep.SetProxy(proxy)
	}

	ep.SetHttpHost("w.apiv3.errplane.com")       // optional (this is the default value)
	ep.SetUdpAddr("udp.apiv3.errplane.com:8126") // optional (this is the default value)

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
	}

	ep.Close()

	time.Sleep(10 * time.Millisecond)

	os.Exit(0)
}
