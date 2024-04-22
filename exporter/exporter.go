package exporter

import (
	"os/exec"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/john-craig/smartctl_ssacli_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
)

// An Exporter is a Prometheus exporter for metrics.
// It wraps all metrics collectors and provides a single global
// exporter which can serve metrics.
//
// It implements the exporter.Collector interface in order to register
// with Prometheus.
type Exporter struct {
	smartctlPath string
	ssacliPath   string
	lsscsiPath   string

	logger log.Logger
}

var _ prometheus.Collector = &Exporter{}

// New creates a new Exporter which collects metrics by creating a apcupsd
// client using the input ClientFunc.
func New(
	logger log.Logger,
	smartctlPath string,
	ssacliPath string,
	lsscsiPath string) *Exporter {
	return &Exporter{
		logger:       logger,
		smartctlPath: smartctlPath,
		ssacliPath:   ssacliPath,
		lsscsiPath:   lsscsiPath}
}

// Describe sends all the descriptors of the collectors included to
// the provided channel.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(e, ch)
}

// Collect sends the collected metrics from each of the collectors to
// exporter.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	level.Debug(e.logger).Log("msg", "Exporter: Collect function called")
	collector.NewSsacliSumCollector(e.logger, e.ssacliPath, e.lsscsiPath).Collect(ch)
	conIDs := collector.ConIDs
	conDevs := collector.ConDevs

	for i := 0; i < len(conIDs); i++ {
		conID := conIDs[i]
		conDev := conDevs[i]

		level.Debug(e.logger).Log("msg", "Exporter: Invoking ssacli binary", "ssacliPath", e.ssacliPath)
		out, err := exec.Command(e.ssacliPath, "ctrl", "slot="+conID, "pd", "all", "show", "status").CombinedOutput()
		level.Info(e.logger).Log("msg", "Exporter: ssacli ctrl slot=N pd all show status", "conId", conID, "out", string(out))

		if err != nil {
			level.Error(e.logger).Log("msg", "Failed collecting metric", "out", out, "err", err)
			return
		}

		physDiskLines := strings.Split(string(out), "\n")
		physDiskN := 0
		for _, physDiskLine := range physDiskLines {
			if strings.TrimSpace(physDiskLine) == "" {
				continue
			}

			physDiskFields := strings.Fields(physDiskLine)
			physDisk := physDiskFields[1]

			collector.NewSsacliPhysDiskCollector(e.logger, physDisk, conID, e.ssacliPath).Collect(ch)
			collector.NewSmartctlDiskCollector(e.logger, physDisk, physDiskN, conDev, e.smartctlPath, ch).Collect(ch)
			physDiskN++
		}

		// Export logic raid status
		out, err = exec.Command(e.ssacliPath, "ctrl", "slot="+conID, "ld", "all", "show", "status").CombinedOutput()

		if err != nil {
			level.Error(e.logger).Log("msg", "Failed collecting metric", "out", out, "err", err)
			return
		}

		logDiskLines := strings.Split(string(out), "\n")
		for _, logDiskLine := range logDiskLines {
			if strings.TrimSpace(logDiskLine) == "" {
				continue
			}

			logDiskFields := strings.Fields(logDiskLine)
			logDisk := logDiskFields[1]

			collector.NewSsacliLogDiskCollector(e.logger, logDisk, conID, e.ssacliPath).Collect(ch)
		}
	}
}
