/*
Copyright paskal.maksim@gmail.com
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/maksim-paskal/envoy-sidecar-helper/pkg/api"
	"github.com/maksim-paskal/envoy-sidecar-helper/pkg/client"
	log "github.com/sirupsen/logrus"
)

var gitVersion = "dev"

var (
	logLevel        = flag.String("log.level", "INFO", "log level (DEBUG, INFO, WARN, ERROR, FATAL, PANIC)")
	logPretty       = flag.Bool("log.pretty", false, "logs in text format")
	logReportCaller = flag.Bool("log.reportCaller", true, "log line number and file name")
)

func main() {
	flag.Parse()

	log.Infof("Staring envoy-sidecar-helper %s...", gitVersion)

	logLevel, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.SetLevel(logLevel)

	if *logPretty {
		log.SetFormatter(&log.TextFormatter{})
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}

	if logLevel == log.DebugLevel || *logReportCaller {
		log.SetReportCaller(true)
	}

	if err := client.Init(); err != nil {
		log.WithError(err).Fatal()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChanInterrupt := make(chan os.Signal, 1)
	signal.Notify(signalChanInterrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChanInterrupt
		log.Error("Got interruption signal...")
		cancel()

		<-signalChanInterrupt
		os.Exit(1)
	}()

	// wait for envoy start
	api.CheckEnvoyStart(ctx)

	// check container status
	api.CheckContainerStop(ctx)
}
