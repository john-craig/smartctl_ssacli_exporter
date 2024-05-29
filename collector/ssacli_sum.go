package collector

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/john-craig/smartctl_ssacli_exporter/parser"
	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &SsacliSumCollector{}

// SsacliSumCollector Contain raid controller detail information
type SsacliSumCollector struct {
	logger log.Logger

	ssacliPath string
	lsscsiPath string
	sudoPath   string

	cachedData  *parser.SsacliSum
	lastCollect time.Time

	ConIDs  []string
	ConDevs []string

	hwConSlotDesc      *prometheus.Desc
	cacheSizeDesc      *prometheus.Desc
	availCacheSizeDesc *prometheus.Desc
	hwConTempDesc      *prometheus.Desc
	cacheModuTempDesc  *prometheus.Desc
	batteryTempDesc    *prometheus.Desc
}

// NewSsacliSumCollector Create new collector
func NewSsacliSumCollector(
	logger log.Logger,
	ssacliPath string,
	lsscsiPath string,
	sudoPath string) *SsacliSumCollector {
	// Init labels
	var (
		namespace = "ssacli"
		subsystem = "hw_raid_controller"
		labels    = []string{
			"raidControllerSN",
			"raidControllerStatus",
			"raidControllerFirmVersion",
			"raidControllerBatteryStatus",
			"raidControllerEncryption",
			"raidControllerDriverName",
			"raidControllerDriverVersion",
		}
	)
	// Return Colected metric to ch <-
	// Include labels
	return &SsacliSumCollector{
		logger: logger,

		ssacliPath: ssacliPath,
		lsscsiPath: lsscsiPath,
		sudoPath:   sudoPath,

		ConIDs:  make([]string, 0),
		ConDevs: make([]string, 0),

		cachedData:  nil,
		lastCollect: time.Now(),

		hwConSlotDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "slot"),
			"Hardware raid controller slot usage",
			labels,
			nil,
		),
		cacheSizeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "cacheSize"),
			"Hardware raid controller total cache size",
			labels,
			nil,
		),
		availCacheSizeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "available_cacheSize"),
			"Hardware raid controller total available cache size",
			labels,
			nil,
		),
		hwConTempDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "temperature"),
			"Hardware raid controller hardware temperature",
			labels,
			nil,
		),
		cacheModuTempDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "temperature_cacheModule"),
			"Hardware raid controller cache module temperature",
			labels,
			nil,
		),
		batteryTempDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "temperature_battery"),
			"Hardware raid controller battery/capacitor module temperature",
			labels,
			nil,
		),
	}
}

// Describe return all description to chanel
func (c *SsacliSumCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *SsacliSumCollector) Collect(ch chan<- prometheus.Metric) {
	level.Debug(c.logger).Log("msg", "SsacliSumCollector: Collect function called")

	data := c.cachedData
	if c.cachedData == nil || time.Now().After(c.lastCollect.Add(time.Minute)) {
		c.ConIDs = make([]string, 0)
		c.ConDevs = make([]string, 0)

		level.Info(c.logger).Log("msg", "SsacliSumCollector: Invoking ssacli binary", "ssacliPath", c.ssacliPath)
		out, err := exec.Command(c.sudoPath, c.ssacliPath, "ctrl", "all", "show", "detail").CombinedOutput()
		level.Debug(c.logger).Log("msg", "SsacliSumCollector: ssacli ctrl all show detail", "out", out)

		if err != nil {
			level.Error(c.logger).Log("msg", "Failed to execute shell command", "out", out)
			return
		}

		data = parser.ParseSsacliSum(string(out))

		for i := range data.SsacliSumData {
			if !slices.Contains(c.ConIDs, data.SsacliSumData[i].SlotID) {
				c.ConIDs = append(c.ConIDs, data.SsacliSumData[i].SlotID)
			}
		}

		// Use the `lsscsi -g` command to determine which controllers
		// correspond to which /dev/sga path
		level.Info(c.logger).Log("msg", "SsacliSumCollector: Invoking lsscsi binary", "lsscsiPath", c.lsscsiPath)
		out, err = exec.Command(c.lsscsiPath, "-g").CombinedOutput()
		level.Debug(c.logger).Log("msg", "SsacliSumCollector: lsscsi -g", "out", out)

		if err != nil {
			level.Error(c.logger).Log("msg", "Failed to execute shell command", "out", out)
			return
		}

		scsiDisks := strings.Split(string(out), "\n")
		for _, scsiDisk := range scsiDisks {
			scsiFields := strings.Fields(scsiDisk)
			if len(scsiFields) != 7 {
				continue
			}

			if scsiFields[1] == "storage" {
				if !slices.Contains(c.ConDevs, scsiFields[6]) {
					c.ConDevs = append(c.ConDevs, scsiFields[6])
				}
			}
		}

		c.cachedData = data
		c.lastCollect = time.Now()
	}

	for i := range data.SsacliSumData {
		var (
			labels = []string{
				data.SsacliSumData[i].SerialNumber,
				data.SsacliSumData[i].ContStatus,
				data.SsacliSumData[i].FirmVersion,
				data.SsacliSumData[i].BatteryStatus,
				data.SsacliSumData[i].Encryption,
				data.SsacliSumData[i].DriverName,
				data.SsacliSumData[i].DriverVersion,
			}
		)

		ch <- prometheus.MustNewConstMetric(
			c.hwConSlotDesc,
			prometheus.GaugeValue,
			float64(data.SsacliSumData[i].Slot),
			labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.cacheSizeDesc,
			prometheus.GaugeValue,
			float64(data.SsacliSumData[i].TotalCacheSize),
			labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.availCacheSizeDesc,
			prometheus.GaugeValue,
			float64(data.SsacliSumData[i].AvailCacheSize),
			labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.hwConTempDesc,
			prometheus.GaugeValue,
			float64(data.SsacliSumData[i].ContTemp),
			labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.cacheModuTempDesc,
			prometheus.GaugeValue,
			float64(data.SsacliSumData[i].CacheModuTemp),
			labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.batteryTempDesc,
			prometheus.GaugeValue,
			float64(data.SsacliSumData[i].BatteryTemp),
			labels...,
		)

	}

	level.Debug(c.logger).Log("msg", "SsacliSumCollector: Collection completed", "data", fmt.Sprintf("%+v", data), "conIDs", fmt.Sprintf("%+v", c.ConIDs), "conDevs", fmt.Sprintf("%+v", c.ConDevs))

	if len(c.ConIDs) != len(c.ConDevs) {
		level.Warn(c.logger).Log("msg", "hpssacli and lsscsi returned different number of controllers")
	}
}
