package collector

import (
	"os/exec"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tidwall/gjson"
)

var _ prometheus.Collector = &SmartctlDiskCollector{}

// SmartctlDiskCollector Contain raid controller detail information
type SmartctlDiskCollector struct {
	logger log.Logger

	smartctlPath string

	ConID  string
	ConDev string
	DiskN  int

	lastCollect time.Time

	embed *SMARTctl
}

// Parse json to gjson object
func parseJSON(data string) gjson.Result {
	if !gjson.Valid(data) {
		return gjson.Parse("{}")
	}
	return gjson.Parse(data)
}

// NewSmartctlDiskCollector Create new collector
func NewSmartctlDiskCollector(
	logger log.Logger,
	conID string,
	conDev string,
	diskN int,
	smartctlPath string) *SmartctlDiskCollector {
	level.Debug(logger).Log("msg", "SmartctlDiskCollector: NewSmartctlDiskCollector function called")

	return &SmartctlDiskCollector{
		logger:       logger,
		ConID:        conID,
		ConDev:       conDev,
		DiskN:        diskN,
		smartctlPath: smartctlPath,
		lastCollect:  time.Now(),
		embed:        nil}
}

// Describe return all description to chanel
func (c *SmartctlDiskCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

// Collect create collector
// Get metric
// Handle error
func (c *SmartctlDiskCollector) Collect(ch chan<- prometheus.Metric) {
	if c.embed == nil || time.Now().After(c.lastCollect.Add(time.Minute)) {
		level.Info(c.logger).Log("msg", "SmartctlDiskCollector: Invoking smartctl binary", "smartctlPath", c.smartctlPath)
		out, err := exec.Command(c.smartctlPath, "--json", "--info", "--health", "--attributes", "--tolerance=verypermissive", "--nocheck=standby", "--all", "-d", "cciss,"+strconv.Itoa(c.DiskN), c.ConDev).CombinedOutput()
		level.Debug(c.logger).Log("msg", "SmartctlDiskCollector: smartctl --info --health --attributes --tolerance=verypermissive --nocheck=standby --all -d ciss,N /dev/sgM", "diskN", strconv.Itoa(c.DiskN), "conDev", c.ConDev, "out", string(out))

		if err != nil {
			level.Error(c.logger).Log("msg", "Failed to execute shell command", "out", string(out))
		}
		json := parseJSON(string(out))

		if c.embed == nil {
			*c.embed = NewSMARTctl(c.logger, json, c.ConID, c.DiskN, ch)
		} else {
			c.embed.json = json
		}

		c.lastCollect = time.Now()
	}

	c.embed.Collect()
}
