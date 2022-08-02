package analyzer

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"metric-collector/grpcs"
	"strconv"
	"sync"
	"time"

	userpb "metric-collector/protos"

	_ "github.com/influxdata/influxdb1-client"
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/sajari/regression"
	"google.golang.org/grpc"
)

var (
	mutex                sync.Mutex
	GPUDegradationCount  int
	NodeDegradationCount int
	DegradationCount     int
	InCreaseCount        int
	FindDegradation      bool
	FindIncrease         bool
)

const portNumber = "9000"

type UserServer struct {
	userpb.UserServer
}

var DegradationPersent = flag.Int("DegradationPersent", 10, "Metric Collect Time")
var CountNumber = flag.Int("CountNumber", 30, "Calculate Metric Number")
var ConfidenceInterval = flag.Float64("ConfidenceInterval", 95, "Confidence Interval(90, 95, 99, 99.9)")

var confidencenum float64

func Analyzer(Node_Metric grpcs.GrpcNode) error {
	flag.Parse()
	if *ConfidenceInterval == 90 {
		confidencenum = 1.645
	} else if *ConfidenceInterval == 95 {
		confidencenum = 1.96
	} else if *ConfidenceInterval == 99 {
		confidencenum = 2.58
	} else {
		if *ConfidenceInterval == 99.9 {
			confidencenum = 3.3
		} else {
			fmt.Println("error Can Not Use This Confidence Interval, So Use Default")
			confidencenum = 1.96
		}
	}
	// fmt.Println(Node_Metric)
	// fmt.Println("degradationpersent : ", *DegradationPersent)
	// fmt.Println(123)
	var Degradation_Data Degradation
	Degradation_Data.NodeName = Node_Metric.GrpcNodeName
	NodeDegradationCount = 0
	GPUDegradationCount = 0
	FindDegradation = false
	FindIncrease = false
	// go debugfunc(&Node_Metric, &Degradation_Data)
	NodeDegradation(&Node_Metric, &Degradation_Data)
	if FindDegradation {
		DegradationCount++
	} else {
		DegradationCount = 0
	}
	if FindIncrease {
		InCreaseCount++
	} else {
		InCreaseCount = 0
	}
	if DegradationCount > 3 || InCreaseCount > 3 {
		degradation_message, err := json.Marshal(Degradation_Data)
		if err != nil {
			fmt.Println("marshal error : ", err)
			return err
		}
		SendDegradationData(string(degradation_message))
	}
	return nil
}

func PodDegradation(c client.Client, Pod_Metric *grpcs.PodMetric, UUID string, Degradation_Pod *DegradationPod) {
	// defer wait_gpu.Done()
	ret := GetPodMetric(c, Pod_Metric, UUID, Degradation_Pod)
	if ret == 0 {
		fmt.Println("Pod Degradation")
		mutex.Lock()
		GPUDegradationCount++
		mutex.Unlock()
	}
}

func GPUDegradation(c client.Client, GPU_Metric *grpcs.GrpcGPU, GPU_Count int, Degradation_GPU *DegradationGPU) {
	// defer wait_node.Done()

	// var wait_gpu sync.WaitGroup
	// wait_gpu.Add(len(GPU_Metric.GPUPod))
	// fmt.Println(GPU_Metric)
	for i := 0; i < len(GPU_Metric.GPUPod); i++ {
		Degradation_GPU.Pod = append(Degradation_GPU.Pod, &DegradationPod{})
		// fmt.Println(Degradation_GPU.Pod)
		// fmt.Println(GPU_Metric.GPUPod[i])
		// fmt.Println(1222)
		// fmt.Println(Degradation_GPU.Pod[i])
		// fmt.Println(i)
		// go PodDegradation(c, GPU_Metric.GPUPod[i], &wait_gpu, GPU_Metric.GrpcGPUUUID, Degradation_GPU.Pod[i])
		go PodDegradation(c, GPU_Metric.GPUPod[i], GPU_Metric.GrpcGPUUUID, Degradation_GPU.Pod[i])
		// fmt.Println(1222 + i)
	}
	// wait_gpu.Wait()
	// if GPUDegradationCount > 2 || GPUDegradationCount == len(GPU_Metric.GPUPod) {
	ret := GetGPUMetrics(c, GPU_Metric, Degradation_GPU)
	if ret == 0 {
		mutex.Lock()
		NodeDegradationCount++
		mutex.Unlock()
		fmt.Println("GPU Degradation")
	}
	// }
}

func NodeDegradation(Node_Metric *grpcs.GrpcNode, Degradation_Data *Degradation) {
	ip := "influxdb.gpu.svc.cluster.local"
	//ip := "10.244.2.2"
	port := "8086"
	url := "http://" + ip + ":" + port
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: url,
	})
	if err != nil {
		fmt.Println("Error creatring influx", err.Error())
	}
	// var wait_node sync.WaitGroup
	// wait_node.Add(len(Node_Metric.NodeGPU))
	for i := 0; i < len(Node_Metric.NodeGPU); i++ {
		Degradation_Data.GPU = append(Degradation_Data.GPU, &DegradationGPU{})
		go GPUDegradation(c, Node_Metric.NodeGPU[i], len(Node_Metric.NodeGPU), Degradation_Data.GPU[i])
		// defer wait.Done()
	}
	// wait_node.Wait()
	c.Close()
	// if NodeDegradationCount > 2 || NodeDegradationCount == len(Node_Metric.NodeGPU) {
	ret := GetNodeMetric(c, Node_Metric, Degradation_Data)
	if ret == 0 {
		fmt.Println("Node Degradation")
	}
	// }

}

func GetNodeMetric(c client.Client, Node_Metric *grpcs.GrpcNode, Degradation_Data *Degradation) int {
	q := client.Query{
		Command:  fmt.Sprintf("SELECT * FROM multimetric where NodeName='%s' order by time desc limit '%d'", Node_Metric.GrpcNodeName, *CountNumber),
		Database: "metric",
	}
	response, err := c.Query(q)
	if err != nil || response.Error() != nil {
		fmt.Println("InfluxDB error: ", err)
		return 2
	}
	if len(response.Results[0].Series[0].Values) < *CountNumber {
		return 2
	}
	for i := 0; i < *CountNumber; i++ {
		if response.Results[0].Series[0].Values[i][8] != Node_Metric.TotalPodnum {
			return 2
		}
	}
	Aver_Metric := NodeMetric{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for i := 0; i < len(response.Results[0].Series[0].Values); i++ {
		myNodeMetric := response.Results[0].Series[0].Values[i]
		AverCPU, _ := strconv.ParseInt(fmt.Sprintf("%s", myNodeMetric[2]), 10, 64)
		AverMemory, _ := strconv.ParseInt(fmt.Sprintf("%s", myNodeMetric[3]), 10, 64)
		AverNetworkTx, _ := strconv.ParseInt(fmt.Sprintf("%s", myNodeMetric[5]), 10, 64)
		AverNetworkRx, _ := strconv.ParseInt(fmt.Sprintf("%s", myNodeMetric[6]), 10, 64)
		AverStorage, _ := strconv.ParseInt(fmt.Sprintf("%s", myNodeMetric[7]), 10, 64)

		Aver_Metric.AverCpu = Aver_Metric.AverCpu + AverCPU
		Aver_Metric.AverMemory = Aver_Metric.AverMemory + AverMemory
		Aver_Metric.AverNetworkRx = Aver_Metric.AverNetworkRx + AverNetworkRx
		Aver_Metric.AverNetworkTx = Aver_Metric.AverNetworkTx + AverNetworkTx
		Aver_Metric.AverStorage = Aver_Metric.AverStorage + AverStorage
	}
	Aver_Metric.AverCpu = Aver_Metric.AverCpu / int64(*CountNumber)
	Aver_Metric.AverMemory = Aver_Metric.AverMemory / int64(*CountNumber)
	Aver_Metric.AverNetworkRx = Aver_Metric.AverNetworkRx / int64(*CountNumber)
	Aver_Metric.AverNetworkTx = Aver_Metric.AverNetworkTx / int64(*CountNumber)
	Aver_Metric.AverStorage = Aver_Metric.AverStorage / int64(*CountNumber)

	for i := 0; i < len(response.Results[0].Series[0].Values); i++ {
		myNodeMetric := response.Results[0].Series[0].Values[i]
		AverCPU, _ := strconv.ParseInt(fmt.Sprintf("%s", myNodeMetric[2]), 10, 64)
		AverMemory, _ := strconv.ParseInt(fmt.Sprintf("%s", myNodeMetric[3]), 10, 64)
		AverNetworkTx, _ := strconv.ParseInt(fmt.Sprintf("%s", myNodeMetric[5]), 10, 64)
		AverNetworkRx, _ := strconv.ParseInt(fmt.Sprintf("%s", myNodeMetric[6]), 10, 64)
		AverStorage, _ := strconv.ParseInt(fmt.Sprintf("%s", myNodeMetric[7]), 10, 64)

		Aver_Metric.StDevCpu += float64((AverCPU - Aver_Metric.AverCpu) ^ 2)
		Aver_Metric.StDevMemory += float64((AverMemory - Aver_Metric.AverMemory) ^ 2)
		Aver_Metric.StDevNetworkRx += float64((AverNetworkRx - Aver_Metric.AverNetworkRx) ^ 2)
		Aver_Metric.StDevNetworkTx += float64((AverNetworkTx - Aver_Metric.AverNetworkTx) ^ 2)
		Aver_Metric.StDevNetworkTx += float64((AverStorage - Aver_Metric.AverStorage) ^ 2)
	}
	Aver_Metric.StDevCpu /= float64(*CountNumber) - 1
	Aver_Metric.StDevMemory /= float64(*CountNumber) - 1
	Aver_Metric.StDevNetworkRx /= float64(*CountNumber) - 1
	Aver_Metric.StDevNetworkTx /= float64(*CountNumber) - 1
	Aver_Metric.StDevStorage /= float64(*CountNumber) - 1

	FindNodeInCrease(Node_Metric, Aver_Metric, Degradation_Data)
	return FindNodeDegradation(Node_Metric, Aver_Metric, Degradation_Data)
}

func GetGPUMetrics(c client.Client, GPU_Metric *grpcs.GrpcGPU, Degradation_GPU *DegradationGPU) int {
	q := client.Query{
		Command:  fmt.Sprintf("select * from gpumetric where UUID='%s' order by time desc limit '%d'", GPU_Metric.GrpcGPUUUID, *CountNumber),
		Database: "metric",
	}
	response, err := c.Query(q)
	if err != nil || response.Error() != nil {
		fmt.Println("InfluxDB error: ", err)
		return 2
	}
	Degradation_GPU.GPUUUID = GPU_Metric.GrpcGPUUUID
	for i := 0; i < *CountNumber; i++ {
		if response.Results[0].Series[0].Values[i][6] != len(GPU_Metric.GPUPod) {
			return 2
		}
	}
	Aver_Metric := GPUMetric{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for i := 0; i < len(response.Results[0].Series[0].Values); i++ {
		myGPUMetric := response.Results[0].Series[0].Values[i]
		AverFan, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[1]))
		AverRX, _ := strconv.ParseInt(fmt.Sprintf("%s", myGPUMetric[4]), 10, 64)
		AverTX, _ := strconv.ParseInt(fmt.Sprintf("%s", myGPUMetric[5]), 10, 64)
		AverPower, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[7]))
		AverMemory, _ := strconv.ParseInt(fmt.Sprintf("%s", myGPUMetric[11]), 10, 64)
		AverTemp, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[12]))
		AverUtil, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[13]))

		Aver_Metric.AverFanSpeed = AverFan + Aver_Metric.AverFanSpeed
		Aver_Metric.AverMemory = AverMemory + Aver_Metric.AverMemory
		Aver_Metric.AverPower = AverPower + Aver_Metric.AverPower
		Aver_Metric.AverRX = AverRX + Aver_Metric.AverRX
		Aver_Metric.AverTX = AverTX + Aver_Metric.AverTX
		Aver_Metric.AverTemp = AverTemp + Aver_Metric.AverTemp
		Aver_Metric.AverUtil = AverUtil + Aver_Metric.AverUtil
	}
	Aver_Metric.AverFanSpeed = Aver_Metric.AverFanSpeed / *CountNumber
	Aver_Metric.AverMemory = Aver_Metric.AverMemory / int64(*CountNumber)
	Aver_Metric.AverPower = Aver_Metric.AverPower / *CountNumber
	Aver_Metric.AverRX = Aver_Metric.AverRX / int64(*CountNumber)
	Aver_Metric.AverTX = Aver_Metric.AverTX / int64(*CountNumber)
	Aver_Metric.AverTemp = Aver_Metric.AverTemp / *CountNumber
	Aver_Metric.AverUtil = Aver_Metric.AverUtil / *CountNumber

	for i := 0; i < len(response.Results[0].Series[0].Values); i++ {
		myGPUMetric := response.Results[0].Series[0].Values[i]
		AverFan, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[1]))
		AverRX, _ := strconv.ParseInt(fmt.Sprintf("%s", myGPUMetric[4]), 10, 64)
		AverTX, _ := strconv.ParseInt(fmt.Sprintf("%s", myGPUMetric[5]), 10, 64)
		AverPower, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[7]))
		AverMemory, _ := strconv.ParseInt(fmt.Sprintf("%s", myGPUMetric[11]), 10, 64)
		AverTemp, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[12]))
		AverUtil, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[13]))

		Aver_Metric.StDevFanSpeed += float64((AverFan - Aver_Metric.AverFanSpeed) ^ 2)
		Aver_Metric.StDevMemory += float64((AverMemory - Aver_Metric.AverMemory) ^ 2)
		Aver_Metric.StDevPower += float64((AverPower - Aver_Metric.AverPower) ^ 2)
		Aver_Metric.StDevRX += float64((AverRX - Aver_Metric.AverRX) ^ 2)
		Aver_Metric.StDevTX += float64((AverTX - Aver_Metric.AverTX) ^ 2)
		Aver_Metric.StDevTemp += float64((AverTemp - Aver_Metric.AverTemp) ^ 2)
		Aver_Metric.StDevUtil += float64((AverUtil - Aver_Metric.AverUtil) ^ 2)
	}
	Aver_Metric.StDevFanSpeed /= float64(*CountNumber) - 1
	Aver_Metric.StDevMemory /= float64(*CountNumber) - 1
	Aver_Metric.StDevPower /= float64(*CountNumber) - 1
	Aver_Metric.StDevRX /= float64(*CountNumber) - 1
	Aver_Metric.StDevTX /= float64(*CountNumber) - 1
	Aver_Metric.StDevTemp /= float64(*CountNumber) - 1
	Aver_Metric.StDevUtil /= float64(*CountNumber) - 1
	FindGPUInCrease(GPU_Metric, Aver_Metric, Degradation_GPU)
	return FindGPUDegradation(GPU_Metric, Aver_Metric, Degradation_GPU)
}

func GetPodMetric(c client.Client, Pod_Metric *grpcs.PodMetric, UUID string, Degradation_Pod *DegradationPod) int {
	q := client.Query{
		Command:  fmt.Sprintf("select * from gpumap where gpu_mps_pod = '%s' and gpu_mps_pid = %d and UUID = '%s and gpu_mps_cpu != 0 and gpu_mps_nodememory != 0' order by time desc limit '%d'", Pod_Metric.PodName, Pod_Metric.PodPid, UUID, *CountNumber),
		Database: "metric",
	}
	response, err := c.Query(q)
	if err != nil || response.Error() != nil {
		fmt.Println("InfluxDB error: ", err)
		return 2
		// return nil
	}
	Aver_Metric := PodMetric{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	Degradation_Pod.PodUID = Pod_Metric.PodUid
	// fmt.Println(111111111)
	// fmt.Println(response)
	if response == nil {
		return 2
	}
	if len(response.Results[0].Series[0].Values) < *CountNumber {
		//total result len is not bigger than 30
		return 2
	}

	for i := 0; i < len(response.Results[0].Series[0].Values); i++ {
		myPodMetric := response.Results[0].Series[0].Values[i]
		// fmt.Println(myPodMetric...)
		AverCPU, _ := strconv.ParseFloat(fmt.Sprintf("%s", myPodMetric[2]), 64)
		AverGPUMemory, _ := strconv.ParseInt(fmt.Sprintf("%s", myPodMetric[3]), 10, 64)
		AverNetworkRx, _ := strconv.ParseInt(fmt.Sprintf("%s", myPodMetric[4]), 10, 64)
		AverNetworkTx, _ := strconv.ParseInt(fmt.Sprintf("%s", myPodMetric[5]), 10, 64)
		AverMemory, _ := strconv.ParseInt(fmt.Sprintf("%s", myPodMetric[6]), 10, 64)
		AverStorage, _ := strconv.ParseInt(fmt.Sprintf("%s", myPodMetric[10]), 10, 64)

		Aver_Metric.AverCPU = AverCPU + Aver_Metric.AverCPU
		Aver_Metric.AverGPUMemory = AverGPUMemory + Aver_Metric.AverGPUMemory
		Aver_Metric.AverNetworkRx = AverNetworkRx + Aver_Metric.AverNetworkRx
		Aver_Metric.AverNetworkTx = AverNetworkTx + Aver_Metric.AverNetworkTx
		Aver_Metric.AverMemory = AverMemory + Aver_Metric.AverMemory
		Aver_Metric.AverStorage = AverStorage + Aver_Metric.AverStorage
	}
	Aver_Metric.AverCPU = Aver_Metric.AverCPU / float64(*CountNumber)
	Aver_Metric.AverGPUMemory = Aver_Metric.AverGPUMemory / int64(*CountNumber)
	Aver_Metric.AverNetworkRx = Aver_Metric.AverNetworkRx / int64(*CountNumber)
	Aver_Metric.AverNetworkTx = Aver_Metric.AverNetworkTx / int64(*CountNumber)
	Aver_Metric.AverMemory = Aver_Metric.AverMemory / int64(*CountNumber)
	Aver_Metric.AverStorage = Aver_Metric.AverStorage / int64(*CountNumber)

	for i := 0; i < len(response.Results[0].Series[0].Values); i++ {
		myPodMetric := response.Results[0].Series[0].Values[i]
		// fmt.Println(myPodMetric...)
		AverCPU, _ := strconv.ParseFloat(fmt.Sprintf("%s", myPodMetric[2]), 64)
		AverGPUMemory, _ := strconv.ParseInt(fmt.Sprintf("%s", myPodMetric[3]), 10, 64)
		AverNetworkRx, _ := strconv.ParseInt(fmt.Sprintf("%s", myPodMetric[4]), 10, 64)
		AverNetworkTx, _ := strconv.ParseInt(fmt.Sprintf("%s", myPodMetric[5]), 10, 64)
		AverMemory, _ := strconv.ParseInt(fmt.Sprintf("%s", myPodMetric[6]), 10, 64)
		AverStorage, _ := strconv.ParseInt(fmt.Sprintf("%s", myPodMetric[10]), 10, 64)

		Aver_Metric.StDevCPU += math.Pow(AverCPU-Aver_Metric.AverCPU, 2)
		Aver_Metric.StDevGPUMemory += float64((AverGPUMemory - Aver_Metric.AverGPUMemory) ^ 2)
		Aver_Metric.StDevMemory += float64((AverMemory - Aver_Metric.AverMemory) ^ 2)
		Aver_Metric.StDevNetworkRx += float64((AverNetworkRx - Aver_Metric.AverNetworkRx) ^ 2)
		Aver_Metric.StDevNetworkTx += float64((AverNetworkTx - Aver_Metric.AverNetworkTx) ^ 2)
		Aver_Metric.StDevStorage += float64((AverStorage - Aver_Metric.AverStorage) ^ 2)
	}

	Aver_Metric.StDevCPU /= float64(*CountNumber) - 1
	Aver_Metric.StDevGPUMemory /= float64(*CountNumber) - 1
	Aver_Metric.StDevMemory /= float64(*CountNumber) - 1
	Aver_Metric.StDevNetworkRx /= float64(*CountNumber) - 1
	Aver_Metric.StDevNetworkTx /= float64(*CountNumber) - 1
	Aver_Metric.StDevStorage /= float64(*CountNumber) - 1

	PodThreshold(Aver_Metric, Pod_Metric, c, UUID)

	FindPodInCrease(Pod_Metric, Aver_Metric, Degradation_Pod)
	return FindPodDegradation(Pod_Metric, Aver_Metric, Degradation_Pod)

}

func FindPodDegradation(curr *grpcs.PodMetric, prev PodMetric, Degradation_Pod *DegradationPod) int {
	// fmt.Println(curr)
	// fmt.Println(prev)
	Degradation_Pod.ISDegradation = true
	if curr.PodCPU < prev.AverCPU-prev.AverCPU/float64(*DegradationPersent) {
		FindDegradation = true
		return 0
	} else if curr.PodGPUMemory < prev.AverGPUMemory-prev.AverGPUMemory/int64(*DegradationPersent) {
		FindDegradation = true
		return 0
	} else if curr.PodMemory < prev.AverMemory-prev.AverMemory/int64(*DegradationPersent) {
		FindDegradation = true
		return 0
	} else if curr.PodNetworkRX < prev.AverNetworkRx-prev.AverNetworkRx/int64(*DegradationPersent) {
		FindDegradation = true
		return 0
	} else if curr.PodNetworkTX < prev.AverNetworkTx-prev.AverNetworkTx/int64(*DegradationPersent) {
		FindDegradation = true
		return 0
	} else if curr.PodStorage < prev.AverStorage-prev.AverStorage/int64(*DegradationPersent) {
		FindDegradation = true
		return 0
	}
	Degradation_Pod.ISDegradation = false
	return 1
}

func FindGPUDegradation(curr *grpcs.GrpcGPU, prev GPUMetric, Degradation_GPU *DegradationGPU) int {
	Degradation_GPU.ISDegradation = true
	if curr.FanSpeed < prev.AverFanSpeed - prev.AverFanSpeed / *DegradationPersent {
		FindDegradation = true
		return 0
	} else if curr.GrpcGPUpower < int(prev.AverPower) + int(prev.AverPower) / *DegradationPersent {
		FindDegradation = true
		return 0
	} else if curr.GrpcGPUtemp < int(prev.AverTemp) + int(prev.AverTemp) / *DegradationPersent {
		FindDegradation = true
		return 0
	} else if curr.GrpcGPUutil < int(prev.AverUtil) + int(prev.AverUtil) / *DegradationPersent {
		FindDegradation = true
		return 0
	} /* else if int(curr.GrpcGPUused) < int(prev.AverMemory) - int(prev.AverMemory) / *DegradationPersent {
		return 0
	}*/
	// else if curr.GPURX < int(prev.AverRX) - int(prev.AverRX) / *DegradationPersent {
	// 	FindDegradation = true
	// 	return 0
	// } else if curr.GPUTX < int(prev.AverTX) - int(prev.AverTX) / *DegradationPersent {
	// 	FindDegradation = true
	// 	return 0
	// }
	Degradation_GPU.ISDegradation = false
	return 1
}

func FindNodeDegradation(curr *grpcs.GrpcNode, prev NodeMetric, Degradation_Data *Degradation) int {
	Degradation_Data.ISDegradation = true
	if curr.GrpcNodeCPU < prev.AverCpu-prev.AverCpu/int64(*DegradationPersent) {
		FindDegradation = true
		return 0
	} else if curr.GrpcNodeMemory < prev.AverMemory-prev.AverMemory/int64(*DegradationPersent) {
		FindDegradation = true
		return 0
	} else if curr.GrpcNodeStorage < prev.AverStorage-prev.AverStorage/int64(*DegradationPersent) {
		FindDegradation = true
		return 0
	} else if curr.NodeNetworkRX < prev.AverNetworkRx-prev.AverNetworkRx/int64(*DegradationPersent) {
		FindDegradation = true
		return 0
	} else if curr.NodeNetworkTX < prev.AverNetworkTx-prev.AverNetworkTx/int64(*DegradationPersent) {
		FindDegradation = true
		return 0
	}
	Degradation_Data.ISDegradation = false
	return 1
}

func FindNodeInCrease(curr *grpcs.GrpcNode, prev NodeMetric, Degradation_Data *Degradation) int {
	Degradation_Data.ISIncrease = true
	if curr.GrpcNodeCPU > prev.AverCpu-prev.AverCpu/int64(*DegradationPersent) {
		FindIncrease = true
		return 0
	} else if curr.GrpcNodeMemory > prev.AverMemory-prev.AverMemory/int64(*DegradationPersent) {
		FindIncrease = true
		return 0
	} else if curr.GrpcNodeStorage > prev.AverStorage-prev.AverStorage/int64(*DegradationPersent) {
		FindIncrease = true
		return 0
	} else if curr.NodeNetworkRX > prev.AverNetworkRx-prev.AverNetworkRx/int64(*DegradationPersent) {
		FindIncrease = true
		return 0
	} else if curr.NodeNetworkTX > prev.AverNetworkTx-prev.AverNetworkTx/int64(*DegradationPersent) {
		FindIncrease = true
		return 0
	}
	Degradation_Data.ISIncrease = false
	return 1
}

func FindGPUInCrease(curr *grpcs.GrpcGPU, prev GPUMetric, Degradation_GPU *DegradationGPU) int {
	Degradation_GPU.ISIncrease = true
	if curr.FanSpeed > prev.AverFanSpeed - prev.AverFanSpeed / *DegradationPersent {
		FindIncrease = true
		return 0
	} else if curr.GrpcGPUpower > int(prev.AverPower) + int(prev.AverPower) / *DegradationPersent {
		FindIncrease = true
		return 0
	} else if curr.GrpcGPUtemp > int(prev.AverTemp) + int(prev.AverTemp) / *DegradationPersent {
		FindIncrease = true
		return 0
	} else if curr.GrpcGPUutil > int(prev.AverUtil) + int(prev.AverUtil) / *DegradationPersent {
		FindIncrease = true
		return 0
	} /* else if int(curr.GrpcGPUused) < int(prev.AverMemory) - int(prev.AverMemory) / *DegradationPersent {
		return 0
	}*/
	// else if curr.GPURX < int(prev.AverRX) - int(prev.AverRX) / *DegradationPersent {
	// 	FindDegradation = true
	// 	return 0
	// } else if curr.GPUTX < int(prev.AverTX) - int(prev.AverTX) / *DegradationPersent {
	// 	FindDegradation = true
	// 	return 0
	// }
	Degradation_GPU.ISIncrease = false
	return 1
}

func FindPodInCrease(curr *grpcs.PodMetric, prev PodMetric, Degradation_Pod *DegradationPod) int {
	// fmt.Println(curr)
	// fmt.Println(prev)
	Degradation_Pod.ISIncrease = true
	if curr.PodCPU > prev.AverCPU-prev.AverCPU/float64(*DegradationPersent) {
		FindIncrease = true
		return 0
	} else if curr.PodGPUMemory > prev.AverGPUMemory-prev.AverGPUMemory/int64(*DegradationPersent) {
		FindIncrease = true
		return 0
	} else if curr.PodMemory > prev.AverMemory-prev.AverMemory/int64(*DegradationPersent) {
		FindIncrease = true
		return 0
	} else if curr.PodNetworkRX > prev.AverNetworkRx-prev.AverNetworkRx/int64(*DegradationPersent) {
		FindIncrease = true
		return 0
	} else if curr.PodNetworkTX > prev.AverNetworkTx-prev.AverNetworkTx/int64(*DegradationPersent) {
		FindIncrease = true
		return 0
	} else if curr.PodStorage > prev.AverStorage-prev.AverStorage/int64(*DegradationPersent) {
		FindIncrease = true
		return 0
	}
	Degradation_Pod.ISIncrease = false
	return 1
}

func SendDegradationData(degradation_message string) error {
	fmt.Println(degradation_message)
	ip := "gpu-scheduler.gpu.svc.cluster.local"
	host := ip + portNumber
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		fmt.Println("grpc conn error : ", err)
		return err
	}
	defer conn.Close()

	grpcClient := userpb.NewUserClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	p, err := grpcClient.SendDegradation(ctx, &userpb.DegradationMessage{DegradationData: degradation_message})
	if err != nil {
		fmt.Println("grpc send error : ", err)
		cancel()
		return err
	}
	p.GetDegradationData()
	cancel()
	return nil
}

// func debugfunc(Node_Metric *grpcs.GrpcNode, Degradation_Data *Degradation) {
// 	// for {
// 	fmt.Println(Node_Metric)
// 	fmt.Println(Degradation_Data)
// 	fmt.Println(Node_Metric.NodeGPU[0])
// 	fmt.Println(Node_Metric.NodeGPU[1])
// 	// fmt.Println(Degradation_Data.GPU)
// 	// }
// }

func PodThreshold(aver PodMetric, curr *grpcs.PodMetric, c client.Client, UUID string) bool {

	if aver.AverCPU-confidencenum*aver.StDevCPU/math.Sqrt(float64(*CountNumber)) < curr.PodCPU {
		if GetPodPattern(c, curr, UUID) {
			return false
		}
	} else if float64(aver.AverGPUMemory)-confidencenum*aver.StDevGPUMemory/math.Sqrt(float64(*CountNumber)) < float64(curr.PodGPUMemory) {
		if GetPodPattern(c, curr, UUID) {
			return false
		}
	} else if float64(aver.AverMemory)-confidencenum*aver.StDevMemory/math.Sqrt(float64(*CountNumber)) < float64(curr.PodMemory) {
		if GetPodPattern(c, curr, UUID) {
			return false
		}
	} else if float64(aver.AverNetworkRx)-confidencenum*aver.StDevNetworkRx/math.Sqrt(float64(*CountNumber)) < float64(curr.PodNetworkRX) {
		if GetPodPattern(c, curr, UUID) {
			return false
		}
	} else if float64(aver.AverNetworkTx)-confidencenum*aver.StDevNetworkTx/math.Sqrt(float64(*CountNumber)) < float64(curr.PodNetworkTX) {
		if GetPodPattern(c, curr, UUID) {
			return false
		}
	} else if float64(aver.AverStorage)-confidencenum*aver.StDevStorage/math.Sqrt(float64(*CountNumber)) < float64(curr.PodStorage) {
		if GetPodPattern(c, curr, UUID) {
			return false
		}
	}
	return true
}

func GPUThreshold(aver GPUMetric, curr *grpcs.GrpcGPU, c client.Client) bool {
	if aver.AverFanSpeed*int(aver.StDevFanSpeed)/int(math.Sqrt(float64(*CountNumber))) < curr.FanSpeed {
		if GetPattern(c) {
			return false
		}
	} else if aver.AverMemory*int64(aver.StDevMemory)/int64(math.Sqrt(float64(*CountNumber))) < curr.GrpcGPUused {
		if GetPattern(c) {
			return false
		}
	} else if aver.AverRX*int64(aver.StDevRX)/int64(math.Sqrt(float64(*CountNumber))) < int64(curr.GPURX) {
		if GetPattern(c) {
			return false
		}
	} else if aver.AverTX*int64(aver.StDevTX)/int64(math.Sqrt(float64(*CountNumber))) < int64(curr.GPUTX) {
		if GetPattern(c) {
			return false
		}
	} else if int64(aver.AverPower)*int64(aver.StDevPower)/int64(math.Sqrt(float64(*CountNumber))) < int64(curr.GrpcGPUpower) {
		if GetPattern(c) {
			return false
		}
	} else if int64(aver.AverTemp)*int64(aver.StDevTemp)/int64(math.Sqrt(float64(*CountNumber))) < int64(curr.GrpcGPUtemp) {
		if GetPattern(c) {
			return false
		}
	} else if int64(aver.AverUtil)*int64(aver.StDevUtil)/int64(math.Sqrt(float64(*CountNumber))) < int64(curr.GrpcGPUutil) {
		if GetPattern(c) {
			return false
		}
	}

	return false
}

func NodeThreshold(aver NodeMetric, curr *grpcs.GrpcNode, c client.Client) bool {
	if aver.AverCpu*int64(aver.StDevCpu)/int64(math.Sqrt(float64(*CountNumber))) < curr.GrpcNodeCPU {
		if GetPattern(c) {
			return false
		}
	} else if aver.AverMemory*int64(aver.StDevMemory)/int64(math.Sqrt(float64(*CountNumber))) < curr.GrpcNodeMemory {
		if GetPattern(c) {
			return false
		}
	} else if aver.AverNetworkRx*int64(aver.StDevNetworkRx)/int64(math.Sqrt(float64(*CountNumber))) < curr.NodeNetworkRX {
		if GetPattern(c) {
			return false
		}
	} else if aver.AverNetworkTx*int64(aver.StDevNetworkTx)/int64(math.Sqrt(float64(*CountNumber))) < curr.NodeNetworkTX {
		if GetPattern(c) {
			return false
		}
	} else if aver.AverStorage*int64(aver.StDevStorage)/int64(math.Sqrt(float64(*CountNumber))) < curr.GrpcNodeStorage {
		if GetPattern(c) {
			return false
		}
	}
	return false
}

func GetPodPattern(c client.Client, curr *grpcs.PodMetric, UUID string) bool {
	q := client.Query{
		Command:  fmt.Sprintf("select * from gpumap where gpu_mps_pod = '%s' and gpu_mps_pid = %d and UUID = '%s' order by time desc ", curr.PodName, curr.PodPid, UUID),
		Database: "metric",
	}
	response, err := c.Query(q)
	if err != nil || response.Error() != nil {
		fmt.Println("InfluxDB error: ", err)
		return false
		// return nil
	}
	var regressiondata Regression
	r := new(regression.Regression)
	r.SetObserved("Dgradation Metric")
	r.SetVar(0, "var1")
	r.SetVar(1, "var2")
	r.SetVar(2, "var3")
	for i := len(response.Results[0].Series[0].Values) - 1; i == 0; i-- {
		myPodMetric := response.Results[0].Series[0].Values[i]
		// fmt.Println(myPodMetric...)
		AverCPU, _ := strconv.ParseFloat(fmt.Sprintf("%s", myPodMetric[2]), 64)
		AverGPUMemory, _ := strconv.ParseFloat(fmt.Sprintf("%s", myPodMetric[3]), 64)
		AverNetworkRx, _ := strconv.ParseFloat(fmt.Sprintf("%s", myPodMetric[4]), 64)
		AverNetworkTx, _ := strconv.ParseFloat(fmt.Sprintf("%s", myPodMetric[5]), 64)
		AverMemory, _ := strconv.ParseFloat(fmt.Sprintf("%s", myPodMetric[6]), 64)
		AverStorage, _ := strconv.ParseFloat(fmt.Sprintf("%s", myPodMetric[10]), 64)

		regressiondata.CPU = append(regressiondata.CPU, AverCPU)
		regressiondata.GPUMemory = append(regressiondata.GPUMemory, AverGPUMemory)
		regressiondata.RX = append(regressiondata.RX, AverNetworkRx)
		regressiondata.TX = append(regressiondata.TX, AverNetworkTx)
		regressiondata.Memory = append(regressiondata.Memory, AverMemory)
		regressiondata.Storage = append(regressiondata.Storage, AverStorage)
	}

	return false
}

func GetPattern(c client.Client) bool {
	return false
}
