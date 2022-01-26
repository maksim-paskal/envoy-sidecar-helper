package main

import (
	"flag"

	"github.com/maksim-paskal/envoy-sidecar-helper/pkg/api"
	"github.com/maksim-paskal/envoy-sidecar-helper/pkg/client"
	log "github.com/sirupsen/logrus"
)

var gitVersion = "dev"

var (
	logLevel        = flag.String("log.level", "INFO", "")
	logPretty       = flag.Bool("log.pretty", false, "")
	logReportCaller = flag.Bool("log.reportCaller", true, "")
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

	// wait for envoy start
	api.CheckEnvoyStart()

	// check container status
	api.CheckContainerStop()
}
