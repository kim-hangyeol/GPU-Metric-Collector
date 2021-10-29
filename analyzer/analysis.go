package analyzer

import (
	"fmt"
	"strconv"

	client "github.com/influxdata/influxdb1-client/v2"
)

func Analyzer(c client.Client, Nodename string, uuids []string) {

	for {
	}

}
func GetNodeMetric(c client.Client, nodeName string, ip string) *NodeMetric {
	q := client.Query{
		Command:  fmt.Sprintf("SELECT last(*) FROM multimetric where NodeName='%s'", nodeName),
		Database: "metric",
	}
	response, err := c.Query(q)
	if err != nil || response.Error() != nil {
		fmt.Println("InfluxDB error: ", err)
		return nil
	}
	myNodeMetric := response.Results[0].Series[0].Values[0]
	totalGPUCount, _ := strconv.Atoi(fmt.Sprintf("%s", myNodeMetric[1]))
	nodeCPU := fmt.Sprintf("%s", myNodeMetric[2])
	nodeMemory := fmt.Sprintf("%s", myNodeMetric[3])
	uuids := stringToArray(myNodeMetric[5].(string))
}

func GetGPUMetrics(c client.Client, uuids []string, ip string) []*GPUMetric {
	var tempGPUMetrics []*GPUMetric
	for _, uuid := range uuids {
		q := client.Query{
			Command:  fmt.Sprintf("SELECT last(*) FROM gpumetric where UUID='%s'", uuid),
			Database: "metric",
		}
		response, err := c.Query(q)
		if err != nil || response.Error() != nil {
			fmt.Println("InfluxDB error: ", err)
			return nil
		}
		myGPUMetric := response.Results[0].Series[0].Values[0]
		gpuName := fmt.Sprintf("%s", myGPUMetric[1])
		mpsIndex, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[2]))
		gpuPower, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[3]))
		gpuMemoryTotal, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[5]))
		gpuMemoryFree, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[4]))
		gpuMemoryUsed, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[6]))
		gpuTemperature, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[7]))
	}
}

func GetPodRecord() {

}

func PodDegradation() {

}

func GPUDegradation() {

}

func NodeDegradation() {

}
