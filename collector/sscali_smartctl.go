package collector

import (
	"os/exec"
	"strconv"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tidwall/gjson"
)

var _ prometheus.Collector = &SmartctlDiskCollector{}

// SmartctlDiskCollector Contain raid controller detail information
type SmartctlDiskCollector struct {
	embed SMARTctl
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
	diskID string,
	diskN int,
	conDev string,
	smartctlPath string,
	ch chan<- prometheus.Metric) *SmartctlDiskCollector {
	level.Debug(logger).Log("msg", "SmartctlDiskCollector: NewSmartctlDiskCollector function called")

	level.Debug(logger).Log("msg", "SmartctlDiskCollector: Invoking smartctl binary", "smartctlPath", smartctlPath)
	out, err := exec.Command(smartctlPath, "--json", "--info", "--health", "--attributes", "--tolerance=verypermissive", "--nocheck=standby", "--all", "-d", "cciss,"+strconv.Itoa(diskN), conDev).CombinedOutput()
	level.Info(logger).Log("msg", "SmartctlDiskCollector: smartctl --info --health --attributes --tolerance=verypermissive --nocheck=standby --all -d ciss,N /dev/sgM", "diskN", strconv.Itoa(diskN), "conDev", conDev, "out", string(out))

	if err != nil {
		level.Error(logger).Log("msg", "Failed to execute shell command", "out", string(out))
		return nil
	}
	json := parseJSON(string(out))

	return &SmartctlDiskCollector{
		embed: NewSMARTctl(logger, json, ch)}
}

// Describe return all description to chanel
func (c *SmartctlDiskCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

// Collect create collector
// Get metric
// Handle error
func (c *SmartctlDiskCollector) Collect(ch chan<- prometheus.Metric) {
	c.embed.Collect()
}
