package collector

import (
	"os/exec"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/john-craig/smartctl_ssacli_exporter/parser"
	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &SsacliLogDiskCollector{}

// SsacliLogDiskCollector Contain raid controller detail information
type SsacliLogDiskCollector struct {
	diskID     string
	conID      string
	ssacliPath string

	logger log.Logger

	cylinders *prometheus.Desc
}

// NewSsacliLogDiskCollector Create new collector
func NewSsacliLogDiskCollector(logger log.Logger, diskID, conID string, ssacliPath string) *SsacliLogDiskCollector {
	// Init labels
	var (
		namespace = "ssacli"
		subsystem = "logical_array"
		labels    = []string{
			"Size",
			"Status",
			"Caching",
			"UID",
			"LName",
			"LID",
		}
	)

	// Rerutn Colected metric to ch <-
	// Include labels
	return &SsacliLogDiskCollector{
		diskID:     diskID,
		conID:      conID,
		ssacliPath: ssacliPath,
		logger:     logger,
		cylinders: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "cylinders"),
			"Logical array cylinder count",
			labels,
			nil,
		),
	}
}

// Describe return all description to chanel
func (c *SsacliLogDiskCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.cylinders,
	}
	for _, d := range ds {
		ch <- d
	}
}

func (c *SsacliLogDiskCollector) Collect(ch chan<- prometheus.Metric) {
	// Export logic raid status
	level.Debug(c.logger).Log("msg", "SsacliLogDiskCollector: Collect function called")

	level.Debug(c.logger).Log("msg", "SsacliLogDiskCollector: Invoking ssacli binary", "ssacliPath", c.ssacliPath)
	out, err := exec.Command(c.ssacliPath, "ctrl", "slot="+c.conID, "ld", c.diskID, "show").CombinedOutput()
	level.Info(c.logger).Log("msg", "SsacliLogDiskCollector: ssacli ctrl slot=N ld M show", "conID", c.conID, "diskID", c.diskID, "out", string(out))

	if err != nil {
		level.Error(c.logger).Log("msg", "Failed to execute shell command", "out", string(out))
		return
	}

	data := parser.ParseSsacliLogDisk(string(out))

	// if data == nil {
	// 	log.Fatal("Unable get data from ssacli logical array exporter")
	// 	return
	// }

	var (
		labels = []string{
			data.SsacliLogDiskData.Size,
			data.SsacliLogDiskData.Status,
			data.SsacliLogDiskData.Caching,
			data.SsacliLogDiskData.UID,
			data.SsacliLogDiskData.LName,
			data.SsacliLogDiskData.LID,
		}
	)

	ch <- prometheus.MustNewConstMetric(
		c.cylinders,
		prometheus.GaugeValue,
		float64(data.SsacliLogDiskData.Cylinders),
		labels...,
	)

	return
}
