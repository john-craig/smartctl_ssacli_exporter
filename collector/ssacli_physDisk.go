package collector

import (
	"os/exec"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/john-craig/smartctl_ssacli_exporter/parser"
	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &SsacliPhysDiskCollector{}

// SsacliPhysDiskCollector Contain raid controller detail information
type SsacliPhysDiskCollector struct {
	logger log.Logger

	diskID     string
	conID      string
	ssacliPath string

	cachedData  *parser.SsacliPhysDisk
	lastCollect time.Time

	curTemp *prometheus.Desc
	maxTemp *prometheus.Desc
}

// NewSsacliPhysDiskCollector Create new collector
func NewSsacliPhysDiskCollector(logger log.Logger, diskID, conID string, ssacliPath string) *SsacliPhysDiskCollector {
	// Init labels
	var (
		namespace = "ssacli"
		subsystem = "physical_disk"
		labels    = []string{
			"diskID",
			"Status",
			"DriveType",
			"IntType",
			"Size",
			"BlockSize",
			"SN",
			"WWID",
			"Model",
			"Bay",
		}
	)

	// Rerutn Colected metric to ch <-
	// Include labels
	return &SsacliPhysDiskCollector{
		logger:     logger,
		diskID:     diskID,
		conID:      conID,
		ssacliPath: ssacliPath,

		cachedData:  nil,
		lastCollect: time.Now(),

		curTemp: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "curTemp"),
			"Actual physical disk temperature",
			labels,
			nil,
		),
		maxTemp: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "maxTmp"),
			"Physical disk maximum temperature",
			labels,
			nil,
		),
	}
}

// Describe return all description to chanel
func (c *SsacliPhysDiskCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *SsacliPhysDiskCollector) Collect(ch chan<- prometheus.Metric) {
	// Export logic raid status
	level.Debug(c.logger).Log("msg", "SsacliPhysDiskCollector: Collect function called")

	data := c.cachedData
	if c.cachedData == nil || time.Now().After(c.lastCollect.Add(time.Minute)) {
		level.Info(c.logger).Log("msg", "SsacliPhysDiskCollector: Invoking ssacli binary", "ssacliPath", c.ssacliPath)
		out, err := exec.Command(c.ssacliPath, "ctrl", "slot="+c.conID, "pd", c.diskID, "show", "detail").CombinedOutput()
		level.Debug(c.logger).Log("msg", "SsacliPhysDiskCollector: ssacli ctrl slot=N pd M show", "conID", c.conID, "diskID", c.diskID, "out", string(out))

		if err != nil {
			level.Error(c.logger).Log("msg", "Failed to execute shell command", "out", string(out))
			return
		}

		data = parser.ParseSsacliPhysDisk(string(out))
		c.cachedData = data
		c.lastCollect = time.Now()
	}

	var (
		labels = []string{
			c.diskID,
			data.SsacliPhysDiskData.Status,
			data.SsacliPhysDiskData.DriveType,
			data.SsacliPhysDiskData.IntType,
			data.SsacliPhysDiskData.Size,
			data.SsacliPhysDiskData.BlockSize,
			data.SsacliPhysDiskData.SN,
			data.SsacliPhysDiskData.WWID,
			data.SsacliPhysDiskData.Model,
			data.SsacliPhysDiskData.Bay,
		}
	)

	ch <- prometheus.MustNewConstMetric(
		c.curTemp,
		prometheus.GaugeValue,
		float64(data.SsacliPhysDiskData.CurTemp),
		labels...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.maxTemp,
		prometheus.GaugeValue,
		float64(data.SsacliPhysDiskData.MaxTemp),
		labels...,
	)
	return
}
