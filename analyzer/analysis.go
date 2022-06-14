package analyzer

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"metric-collector/grpcs"
	"strconv"
	"sync"
	"time"

	userpb "metric-collector/protos"

	_ "github.com/influxdata/influxdb1-client"
	client "github.com/influxdata/influxdb1-client/v2"
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

func Analyzer(Node_Metric grpcs.GrpcNode) error {
	flag.Parse()
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
		Command:  fmt.Sprintf("SELECT * FROM multimetric where NodeName='%s' order by time desc limit 30", Node_Metric.GrpcNodeName),
		Database: "metric",
	}
	response, err := c.Query(q)
	if err != nil || response.Error() != nil {
		fmt.Println("InfluxDB error: ", err)
		return 2
	}
	if len(response.Results[0].Series[0].Values) < 30 {
		return 2
	}
	for i := 0; i < 30; i++ {
		if response.Results[0].Series[0].Values[i][8] != Node_Metric.TotalPodnum {
			return 2
		}
	}
	Aver_Metric := NodeMetric{0, 0, 0, 0, 0}
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
	Aver_Metric.AverCpu = Aver_Metric.AverCpu / 30
	Aver_Metric.AverMemory = Aver_Metric.AverMemory / 30
	Aver_Metric.AverNetworkRx = Aver_Metric.AverNetworkRx / 30
	Aver_Metric.AverNetworkTx = Aver_Metric.AverNetworkTx / 30
	Aver_Metric.AverStorage = Aver_Metric.AverStorage / 30

	FindNodeInCrease(Node_Metric, Aver_Metric, Degradation_Data)
	return FindNodeDegradation(Node_Metric, Aver_Metric, Degradation_Data)
}

func GetGPUMetrics(c client.Client, GPU_Metric *grpcs.GrpcGPU, Degradation_GPU *DegradationGPU) int {
	q := client.Query{
		Command:  fmt.Sprintf("select * from gpumetric where UUID='%s' order by time desc limit 30", GPU_Metric.GrpcGPUUUID),
		Database: "metric",
	}
	response, err := c.Query(q)
	if err != nil || response.Error() != nil {
		fmt.Println("InfluxDB error: ", err)
		return 2
	}
	Degradation_GPU.GPUUUID = GPU_Metric.GrpcGPUUUID
	for i := 0; i < 30; i++ {
		if response.Results[0].Series[0].Values[i][6] != len(GPU_Metric.GPUPod) {
			return 2
		}
	}
	Aver_Metric := GPUMetric{0, 0, 0, 0, 0, 0, 0}
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
	Aver_Metric.AverFanSpeed = Aver_Metric.AverFanSpeed / 30
	Aver_Metric.AverMemory = Aver_Metric.AverMemory / 30
	Aver_Metric.AverPower = Aver_Metric.AverPower / 30
	Aver_Metric.AverRX = Aver_Metric.AverRX / 30
	Aver_Metric.AverTX = Aver_Metric.AverTX / 30
	Aver_Metric.AverTemp = Aver_Metric.AverTemp / 30
	Aver_Metric.AverUtil = Aver_Metric.AverUtil / 30

	FindGPUInCrease(GPU_Metric, Aver_Metric, Degradation_GPU)
	return FindGPUDegradation(GPU_Metric, Aver_Metric, Degradation_GPU)
}

func GetPodMetric(c client.Client, Pod_Metric *grpcs.PodMetric, UUID string, Degradation_Pod *DegradationPod) int {
	q := client.Query{
		Command:  fmt.Sprintf("select * from gpumap where gpu_mps_pod = '%s' and gpu_mps_pid = %d and UUID = '%s' order by time desc limit 30", Pod_Metric.PodName, Pod_Metric.PodPid, UUID),
		Database: "metric",
	}
	response, err := c.Query(q)
	if err != nil || response.Error() != nil {
		fmt.Println("InfluxDB error: ", err)
		return 2
		// return nil
	}
	Aver_Metric := PodMetric{0, 0, 0, 0, 0, 0}
	Degradation_Pod.PodUID = Pod_Metric.PodUid
	// fmt.Println(111111111)
	// fmt.Println(response)
	if response == nil {
		return 2
	}
	if len(response.Results[0].Series[0].Values) < 30 {
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
	Aver_Metric.AverCPU = Aver_Metric.AverCPU / 30
	Aver_Metric.AverGPUMemory = Aver_Metric.AverGPUMemory / 30
	Aver_Metric.AverNetworkRx = Aver_Metric.AverNetworkRx / 30
	Aver_Metric.AverNetworkTx = Aver_Metric.AverNetworkTx / 30
	Aver_Metric.AverMemory = Aver_Metric.AverMemory / 30
	Aver_Metric.AverStorage = Aver_Metric.AverStorage / 30

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
