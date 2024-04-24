package main

import (
	"flag"
	"net/http"

	"github.com/go-kit/log/level"
	"github.com/john-craig/smartctl_ssacli_exporter/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
)

var (
	listenAddr  = flag.String("web.listen-address", ":9633", "Address for exporter")
	metricsPath = flag.String("web.telemetry-path", "/metrics", "URL path for surfacing collected metrics")

	smartctlPath = flag.String("smartctl.path", "/usr/bin/smartctl", "Path to smartctl binary")
	ssacliPath   = flag.String("ssacli.path", "/usr/bin/ssacli", "Path to ssacli binary")
	lsscsiPath   = flag.String("lsscsi.path", "/usr/bin/lsscsci", "Path to lsscsi binary")

	logLevel = flag.NewFlagSet("log.level", flag.ContinueOnError).String("log", "info", "debug, info, warn, error")
)

func main() {
	flag.Parse()

	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)
	logger = level.NewFilter(logger, level.Allow(level.ParseDefault(*logLevel, level.InfoValue())))

	prometheus.MustRegister(exporter.New(logger, *smartctlPath, *ssacliPath, *lsscsiPath))

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsPath, http.StatusMovedPermanently)
	})

	level.Info(logger).Log("msg", "Beginning to serve exporter", "port", *listenAddr, "metricsPath", *metricsPath)

	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		level.Error(logger).Log("msg", "Cannot start exporter", "err", err)
	}
}
