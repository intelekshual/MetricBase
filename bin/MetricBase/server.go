package main

import (
	"fmt"
	"github.com/msiebuhr/MetricBase/backends"
	"github.com/msiebuhr/MetricBase/frontends"
	"github.com/msiebuhr/MetricBase/serverBuilder"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Create server
	mb := serverBuilder.CreateMetricServer()

	// Create and add front- and back-ends

	mb.AddFrontend(frontends.CreateHttpServer("./http-pub"))
	mb.AddFrontend(frontends.CreateGraphiteTcpServer())

	mb.AddBackend(backends.CreateMemoryBackend())
	//mb.AddBackend(backends.CreateLevelDb("./level-db"))

	go mb.Start()

	// Listen for signals and stop
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Stopping server:", <-ch)
	mb.Stop()
}