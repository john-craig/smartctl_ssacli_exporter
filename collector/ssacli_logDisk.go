package collector

import (
	"os/exec"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/john-craig/smartctl_ssacli_exporter/parser"
	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &SsacliLogDiskCollector{}

// SsacliLogDiskCollector Contain raid controller detail information
type SsacliLogDiskCollector struct {
	logger log.Logger

	DiskID     string
	ConID      string
	ssacliPath string

	cachedData  *parser.SsacliLogDisk
	lastCollect time.Time

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
		logger:      logger,
		DiskID:      diskID,
		ConID:       conID,
		ssacliPath:  ssacliPath,
		cachedData:  nil,
		lastCollect: time.Now(),
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
	prometheus.DescribeByCollect(c, ch)
}

func (c *SsacliLogDiskCollector) Collect(ch chan<- prometheus.Metric) {
	// Export logic raid status
	level.Debug(c.logger).Log("msg", "SsacliLogDiskCollector: Collect function called")

	data := c.cachedData
	if c.cachedData == nil || time.Now().After(c.lastCollect.Add(time.Minute)) {
		level.Info(c.logger).Log("msg", "SsacliLogDiskCollector: Invoking ssacli binary", "ssacliPath", c.ssacliPath)
		out, err := exec.Command(c.ssacliPath, "ctrl", "slot="+c.ConID, "ld", c.DiskID, "show").CombinedOutput()
		level.Debug(c.logger).Log("msg", "SsacliLogDiskCollector: ssacli ctrl slot=N ld M show", "conID", c.ConID, "diskID", c.DiskID, "out", string(out))

		if err != nil {
			level.Error(c.logger).Log("msg", "Failed to execute shell command", "out", string(out))
			return
		}

		data = parser.ParseSsacliLogDisk(string(out))
		c.cachedData = data
		c.lastCollect = time.Now()
	}

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
}
