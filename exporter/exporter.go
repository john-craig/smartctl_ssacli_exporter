package exporter

import (
	"os/exec"
	"reflect"
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

	sumCol   collector.SsacliSumCollector
	physCols []collector.SsacliPhysDiskCollector
	logCols  []collector.SsacliLogDiskCollector
	smrtCols []collector.SmartctlDiskCollector

	conIDs  []string
	conDevs []string

	cachedPhysDiskLines [][]string
	cachedLogDiskLines  [][]string

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

	sumCol := collector.NewSsacliSumCollector(logger, ssacliPath, lsscsiPath)

	return &Exporter{
		logger: logger,

		sumCol:   *sumCol,
		physCols: make([]collector.SsacliPhysDiskCollector, 0),
		logCols:  make([]collector.SsacliLogDiskCollector, 0),
		smrtCols: make([]collector.SmartctlDiskCollector, 0),

		conIDs:  make([]string, 0),
		conDevs: make([]string, 0),

		cachedPhysDiskLines: make([][]string, 0),
		cachedLogDiskLines:  make([][]string, 0),

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
	e.sumCol.Collect(ch)
	conIDs := e.sumCol.ConIDs
	conDevs := e.sumCol.ConDevs

	if !reflect.DeepEqual(e.conIDs, conIDs) || !reflect.DeepEqual(e.conDevs, conDevs) {
		// If the controllers changed, fix 'em
		e.physCols = make([]collector.SsacliPhysDiskCollector, 0)
		e.logCols = make([]collector.SsacliLogDiskCollector, 0)
		e.smrtCols = make([]collector.SmartctlDiskCollector, 0)

		e.cachedLogDiskLines = make([][]string, len(e.conIDs))
		e.cachedPhysDiskLines = make([][]string, len(e.conIDs))

		e.conIDs = conIDs
		e.conDevs = conDevs
	}

	for i := 0; i < len(conIDs); i++ {
		conID := conIDs[i]
		conDev := conDevs[i]

		level.Info(e.logger).Log("msg", "Exporter: Invoking ssacli binary", "ssacliPath", e.ssacliPath)
		out, err := exec.Command(e.ssacliPath, "ctrl", "slot="+conID, "pd", "all", "show", "status").CombinedOutput()
		level.Debug(e.logger).Log("msg", "Exporter: ssacli ctrl slot=N pd all show status", "conId", conID, "out", out)

		if err != nil {
			level.Error(e.logger).Log("msg", "Failed collecting metric", "out", out, "err", err)
			return
		}

		physDiskLines := strings.Split(string(out), "\n")

		// For the first time through, e.cachedPhysDiskLines will be empty,
		// so we need to populate it
		if i >= len(e.cachedPhysDiskLines) {
			e.cachedPhysDiskLines = append(e.cachedPhysDiskLines, make([]string, 0))
		}

		if !reflect.DeepEqual(e.cachedPhysDiskLines[i], physDiskLines) {
			e.cachedPhysDiskLines[i] = physDiskLines

			physDiskN := 0
			for _, physDiskLine := range physDiskLines {
				if strings.TrimSpace(physDiskLine) == "" {
					continue
				}

				physDiskFields := strings.Fields(physDiskLine)
				physDisk := physDiskFields[1]

				if !physDiskCollectorExists(e.physCols, physDisk, conID) {
					e.physCols = append(e.physCols, *collector.NewSsacliPhysDiskCollector(e.logger, physDisk, conID, e.ssacliPath))
				}

				if !smartCollectorExists(e.smrtCols, conID, conDev, physDiskN) {
					e.smrtCols = append(e.smrtCols, *collector.NewSmartctlDiskCollector(e.logger, conID, conDev, physDiskN, e.smartctlPath))
				}

				physDiskN++
			}
		}

		// Export logic raid status
		level.Info(e.logger).Log("msg", "Exporter: Invoking ssacli binary", "ssacliPath", e.ssacliPath)
		out, err = exec.Command(e.ssacliPath, "ctrl", "slot="+conID, "ld", "all", "show", "status").CombinedOutput()
		level.Debug(e.logger).Log("msg", "Exporter: ssacli ctrl slot=N ld all show status", "conId", conID, "out", out)

		if err != nil {
			level.Error(e.logger).Log("msg", "Failed collecting metric", "out", out, "err", err)
			return
		}

		// For the first time through, e.cachedLogDiskLines will be empty,
		// so we need to populate it
		if i >= len(e.cachedLogDiskLines) {
			e.cachedLogDiskLines = append(e.cachedLogDiskLines, make([]string, 0))
		}

		logDiskLines := strings.Split(string(out), "\n")
		if reflect.DeepEqual(e.cachedLogDiskLines[i], logDiskLines) {
			e.cachedLogDiskLines[i] = logDiskLines

			for _, logDiskLine := range logDiskLines {
				if strings.TrimSpace(logDiskLine) == "" {
					continue
				}

				logDiskFields := strings.Fields(logDiskLine)
				logDisk := logDiskFields[1]

				if !logDiskCollectorExists(e.logCols, logDisk, conID) {
					e.logCols = append(e.logCols, *collector.NewSsacliLogDiskCollector(e.logger, logDisk, conID, e.ssacliPath))
				}
			}
		}
	}

	// Now collect metrics
	for _, physCol := range e.physCols {
		physCol.Collect(ch)
	}

	for _, smrtCol := range e.smrtCols {
		smrtCol.Collect(ch)
	}

	for _, logCol := range e.logCols {
		logCol.Collect(ch)
	}

}

func physDiskCollectorExists(s []collector.SsacliPhysDiskCollector, diskID string, conID string) bool {
	for _, a := range s {
		if a.DiskID == diskID && a.ConID == conID {
			return true
		}
	}
	return false
}

func logDiskCollectorExists(s []collector.SsacliLogDiskCollector, diskID string, conID string) bool {
	for _, a := range s {
		if a.DiskID == diskID && a.ConID == conID {
			return true
		}
	}
	return false
}

func smartCollectorExists(s []collector.SmartctlDiskCollector, conDev string, conID string, diskN int) bool {
	for _, a := range s {
		if a.ConDev == conDev && a.ConID == conID && a.DiskN == diskN {
			return true
		}
	}
	return false
}
