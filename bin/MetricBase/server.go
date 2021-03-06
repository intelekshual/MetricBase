package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msiebuhr/MetricBase/serverBuilder"

	"github.com/msiebuhr/MetricBase/backends/boltdb"
	"github.com/msiebuhr/MetricBase/backends/testProxy"

	"github.com/msiebuhr/MetricBase/frontends"
	"github.com/msiebuhr/MetricBase/frontends/graphiteTcp"
	"github.com/msiebuhr/MetricBase/frontends/http"
	"github.com/msiebuhr/MetricBase/frontends/internalMetrics"
)

var staticRoot = flag.String("http-pub", "./http-pub", "HTTP public dir")
var boltDb = flag.String("boltdb", "./bolt.db", "Bolt db file")

func main() {
	// Parse command line flags
	flag.Parse()

	// Create backend + database
	bdb, err := boltdb.NewBoltBackend(*boltDb)
	if err != nil {
		fmt.Println("Could not create bolt database", err)
		return
	}

	// Create server
	mb := serverBuilder.NewMetricServer(
		[]frontends.Frontend{
			http.NewHttpServer(*staticRoot),
			graphiteTcp.NewGraphiteTcpServer(),
			internalMetrics.NewInternalMetrics(time.Second),
		},
		//backends.NewTestProxy(backends.NewMemoryBackend()),
		testProxy.NewTestProxy(bdb),
	)

	go mb.Start()

	// Listen for signals and stop
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Stopping server:", <-ch)
	mb.Stop()
}
