package collector

import (
	"os/exec"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/john-craig/smartctl_ssacli_exporter/parser"
	"github.com/prometheus/client_golang/prometheus"
)

var ConIDs []string
var ConDevs []string

var _ prometheus.Collector = &SsacliSumCollector{}

// SsacliSumCollector Contain raid controller detail information
type SsacliSumCollector struct {
	id                 string
	ssacliPath         string
	lsscsiPath         string
	logger             log.Logger
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
	lsscsiPath string) *SsacliSumCollector {
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

	level.Debug(c.logger).Log("msg", "SsacliSumCollector: Invoking ssacli binary", "ssacliPath", c.ssacliPath)
	out, err := exec.Command(c.ssacliPath, "ctrl", "all", "show", "detail").CombinedOutput()
	level.Info(c.logger).Log("msg", "SsacliSumCollector: ssacli ctrl all show detail", "out", string(out))

	if err != nil {
		level.Error(c.logger).Log("msg", "Failed to execute shell command", "out", string(out))
		return
	}

	data := parser.ParseSsacliSum(string(out))

	// if data == nil {
	// 	log.Fatal("Unable get data from ssacli sumarry exporter")
	// 	return nil, nil
	// }

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

		ConIDs = append(ConIDs, data.SsacliSumData[i].SlotID)

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

	// Use the `lsscsi -g` command to determine which controllers
	// correspond to which /dev/sga path
	level.Debug(c.logger).Log("msg", "SsacliSumCollector: Invoking lsscsi binary", "lsscsiPath", c.lsscsiPath)
	out, err = exec.Command(c.lsscsiPath, "-g").CombinedOutput()
	level.Info(c.logger).Log("msg", "SsacliSumCollector: lsscsi -g", "out", string(out))

	if err != nil {
		level.Error(c.logger).Log("msg", "Failed to execute shell command", "out", string(out))
		return
	}

	scsiDisks := strings.Split(string(out), "\n")
	for _, scsiDisk := range scsiDisks {
		scsiFields := strings.Fields(scsiDisk)
		if scsiFields[1] == "storage" {
			ConDevs = append(ConDevs, scsiFields[6])
		}
	}

	if len(ConIDs) != len(ConDevs) {
		level.Warn(c.logger).Log("msg", "hpssacli and lsscsi returned different number of controllers")
	}
}
