package parser

import (
	"strings"
)

// SsacliSum data structure for output
type SsacliSum struct {
	ContNumber    int
	SsacliSumData []SsacliSumData
}

// SsacliSumData data structure for output
type SsacliSumData struct {
	Slot           int64
	SlotID         string
	SerialNumber   string
	ContStatus     string
	FirmVersion    string
	TotalCacheSize float64
	AvailCacheSize float64
	BatteryStatus  string
	ContTemp       float64
	CacheModuTemp  float64
	BatteryTemp    float64
	Encryption     string
	DriverName     string
	DriverVersion  string
}

// ParseSsacliSum return specific metric
func ParseSsacliSum(s string) *SsacliSum {
	data := parseSmartAttrs(s)

	return data
}

func parseSmartAttrs(s string) *SsacliSum {

	var (
		contNumber int
		sumData    []SsacliSumData
	)

	contNumber = 0

	for _, line := range strings.Split(s, "\n") {
		kvs := strings.Trim(line, " \t")
		if kvs == "" {
			continue
		}

		kv := strings.Split(kvs, ": ")

		if len(kv) == 1 {
			sumData = append(sumData, *new(SsacliSumData))
			contNumber++
		} else {
			if len(sumData) != contNumber {
				// We go out of whack somehow: todo, error logging
				break
			}

			switch kv[0] {
			case "Slot":
				sumData[contNumber-1].Slot = toINT(kv[1])
				sumData[contNumber-1].SlotID = kv[1]
			case "Serial Number":
				sumData[contNumber-1].SerialNumber = kv[1]
			case "Controller Status":
				sumData[contNumber-1].ContStatus = kv[1]
			case "Firmware Version":
				sumData[contNumber-1].FirmVersion = kv[1]
			case "Total Cache Size":
				sumData[contNumber-1].TotalCacheSize = toFLO(kv[1])
			case "Total Cache Memory Available":
				sumData[contNumber-1].AvailCacheSize = toFLO(kv[1])
			case "Battery/Capacitor Status":
				sumData[contNumber-1].BatteryStatus = kv[1]
			case "Controller Temperature (C)":
				sumData[contNumber-1].ContTemp = toFLO(kv[1])
			case "Cache Module Temperature (C)":
				sumData[contNumber-1].CacheModuTemp = toFLO(kv[1])
			case "Capacitor Temperature  (C)":
				sumData[contNumber-1].BatteryTemp = toFLO(kv[1])
			case "Encryption":
				sumData[contNumber-1].Encryption = kv[1]
			case "Driver Name":
				sumData[contNumber-1].DriverName = kv[1]
			case "Driver Version":
				sumData[contNumber-1].DriverVersion = kv[1]
			}

		}
	}

	data := SsacliSum{
		ContNumber:    contNumber,
		SsacliSumData: sumData,
	}
	return &data
}
