package gpumetric

import (
	"fmt"
	"log"
	"metric-collector/analyzer"
	"metric-collector/gpumpsmetric"
	"metric-collector/grpcs"
	"metric-collector/storage"
	"time"

	//"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"

	_ "github.com/influxdata/influxdb1-client"
	influxdb "github.com/influxdata/influxdb1-client/v2"

	//"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
	//"nvml-test/gpumpsmetriccollector"
	//"metric-collector/metricfactory"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	//"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
	//_ "github.com/influxdata/influxdb1-client"
	//influxdb "github.com/influxdata/influxdb1-client/v2"
)

//var txnum = 0
//var rxnum = 0
/*type nvml1 struct {
	Init           func() error
	Shutdown       func() error
	GetDeviceCount func() (uint, error)

	NewDevice func(uint) (Device, error)
}
type Device struct {
	Clocks      int
	UUID        string
	Path        string
	Model       string
	Power       uint
	Memory      uint64
	CPUAffinity uint
	Status      func() (Stat, error)
	// contains filtered or unexported fields
}

type Stat struct {
	Memory      int
	Power       int
	Temperature int
	Clocks      int
	PCI         PCIStatusInfo
}

type PCIStatusInfo struct {
	BAR1Used   uint
	Throughput PCIThroughputInfo
}

type PCIThroughputInfo struct {
	RX uint
	TX uint
}*/

func Gpumetric(c influxdb.Client, nodecpu int64, nodememory int64, nodename string, GPU []*grpcs.GrpcGPU, data *storage.Collection, Node *grpcs.GrpcNode) {
	//func Gpumetric(c influxdb.Client, nvml *nvml1) {
	//name := os.Getenv("MY_NODE_NAME")

	//fmt.Printf("nodename: %v\n", name)
	var podmap []*storage.GPUMAP
	var gpuuuid []string
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		//log.Fatalf("Unable to initialize NVML: %v", ret)
		fmt.Printf("Unable to initialize NVML: %v\n", ret)
		bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database:  "metric",
			Precision: "s",
		})

		tags := map[string]string{}
		fields := map[string]interface{}{
			"NodeCPU":    nodecpu,
			"NodeMemory": nodememory,
			"uuid":       gpuuuid,
			"Count":      0,
			"NodeName":   nodename,
		}
		pt, err := influxdb.NewPoint("multimetric", tags, fields, time.Now())
		if err != nil {
			fmt.Println("Error:", err.Error())
		}
		bp.AddPoint(pt)
		err = c.Write(bp)
		if err != nil {
			fmt.Println("Error:", err.Error())
		}
		return
	}
	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			//log.Fatalf("Unable to shutdown NVML: %v", ret)
			fmt.Printf("Unable to shutdown NVML: %v\n", ret)
		}
	}()
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		//log.Fatalf("Unable to get device count: %v", ret)
		fmt.Printf("Unable to get device count: %v", ret)
		count = 0
	}
	//fmt.Printf("GPU Count : %v\n", count)
	//fmt.Printf("GPU Metric Collector log\n")
	//fmt.Println("GPU Index   GPU Utilization   GPU Memory(Total, Free, Used)   GPU Temperature     GPU Power(Total, Used)")
	for i := 0; i < count; i++ {
		if len(podmap) < int(count) {
			podmap = append(podmap, &storage.GPUMAP{})
		}
		//var slurmjob []storage.SlurmJob
		//singularity_test,gpu:1(IDX:0),1649389685
		//idx:0-1,3-5,7
		//gpuworkdata := strings.Split(workdata, " ")
		//fmt.Println(gpuworkdata)
		//fmt.Println(len(gpuworkdata))
		// jobnum := 0
		// for j := 0; j < len(gpuworkdata)-1; j++ {
		// 	onework := strings.Split(gpuworkdata[j], "/")
		// 	gpuuses := onework[1][strings.LastIndexAny(onework[1], ":")+1 : len(onework[1])-1]
		// 	findgpu := strings.Split(gpuuses, ",")
		// 	for k := 0; k < len(findgpu); k++ {
		// 		value := strings.Split(findgpu[k], "-")
		// 		if len(value) == 1 {
		// 			lv, _ := strconv.Atoi(value[0])
		// 			if lv == i {
		// 				slurmjob = append(slurmjob, storage.SlurmJob{})
		// 				slurmjob[jobnum].Index = i
		// 				slurmjob[jobnum].JobName = onework[0]
		// 				slurmjob[jobnum].StartTime = onework[2]
		// 				jobnum++
		// 				break
		// 			}
		// 		} else {
		// 			lv, _ := strconv.Atoi(value[0])
		// 			rv, _ := strconv.Atoi(value[1])
		// 			//fmt.Println(lv, rv)
		// 			if lv <= i && rv >= i {
		// 				slurmjob = append(slurmjob, storage.SlurmJob{})
		// 				slurmjob[jobnum].Index = i
		// 				slurmjob[jobnum].JobName = onework[0]
		// 				slurmjob[jobnum].StartTime = onework[2]
		// 				jobnum++
		// 				break
		// 			}
		// 		}
		// 	}
		// }
		//fmt.Println(slurmjob)

		//fmt.print("13")
		//t := time.Now()
		//fmt.Println(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
		//fmt.Printf("GPU Index : %v\n", i)
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get device at index %d: %v", i, ret)
		}
		// status, ret := device.Status()
		// if ret != nil {
		// 	log.Fatalf("Unable to get status at index %d: %v", i, ret)
		// }
		uuid, _ := device.GetUUID() //uuid
		podmap[i].GPUUUID = uuid
		Node.NodeGPU[i].GPUPod = nil
		podnum := gpumpsmetric.Gpumpsmetric(device, int(count), c, data, podmap[i].PodMetric, Node.NodeGPU[i])

		//process := status.Processes
		//fmt.Printf("%v \n",process)

		//uuid := device.UUID //uuid
		// fmt.Printf("GPU UUID : %v\n", uuid)

		dname, _ := device.GetName() //gpuname
		//fmt.Printf("%v\n", *dname)
		// fmt.Printf("GPU 이름 : %v\n", dname)

		util, _ := device.GetUtilizationRates()
		// fmt.Printf("GPU 사용률(%%) : %v\n", util.Gpu)

		memory, _ := device.GetMemoryInfo() //메모리 정보 total
		//fmt.Printf("%v\n", *memory)
		// fmt.Printf("메모리 총량(Byte)) : %v\n", memory.Total)
		// fmt.Printf("메모리 사용량(Byte) : %v\n", memory.Used)

		// Memory := status.Memory //메모리 정보 used
		//fmt.Printf("메모리 사용량(MiB) (Used, Free) : %v %v\n", *(Memory.Global.Used), *(Memory.Global.Free)) //Memory.ECCErrorsInfo.L1(L2)Cache(Device)
		//fmt.Printf("메모리 사용량 : %v\n", Memory)

		//clock := device.Clocks //클럭정보
		//fmt.Printf("클럭 총량 (Cores, Memory) %v %v\n", *(clock.Cores), *(clock.Memory))
		//fmt.Printf("클럭 총량 : %v\n", clock)

		//Clock := status.Clocks //클럭정보
		//fmt.Printf("클럭 사용량 (Cores, Memory) : %v %v\n", *(Clock.Cores), *(Clock.Memory))
		//fmt.Printf("클럭 사용량 : %v\n", Clock)

		power, _ := device.GetPowerManagementLimit() // 파워 총량
		// fmt.Printf("%v\n", power)
		// fmt.Printf("파워 총량(W) : %v\n", power/1000)

		Power, _ := device.GetPowerUsage() // 파워 사용량
		// fmt.Printf("%v\n", Power)
		// fmt.Printf("파워 사용량(W) : %v\n", Power/1000)

		temperature, _ := device.GetTemperature(0) //온도
		// fmt.Printf("%v\n", temperature)
		// fmt.Printf("현재 온도(°C) : %v\n", temperature)

		// threadholdtemp, _ := device.GetTemperatureThreshold(1)
		// fmt.Printf("성능 저하 온도(°C) : %v\n", threadholdtemp)

		// shutdowntemp, _ := device.GetTemperatureThreshold(0)
		// fmt.Printf("셧다운 온도(°C) : %v\n", shutdowntemp)
		/*pci := status.PCI
		fmt.Printf("bar1 memory 사용량 : %v\n", pci.BAR1Used)

		fmt.Printf("RX, TX 사용량 : %v %v\n", pci.Throughput.RX, pci.Throughput.TX)*/

		//pci := device.PCI //pci
		//fmt.Printf("PCI (BusID, BAR1 Memory, BandWidth): %v %v %v\n", pci.BusID, *(pci.BAR1), *(pci.Bandwidth))
		PCITX, _ := device.GetPcieThroughput(0)
		PCIRX, _ := device.GetPcieThroughput(1)
		fanspeed, _ := device.GetFanSpeed()

		gpuuuid = append(gpuuuid, uuid)

		GPU[i].GrpcGPUused = int64(memory.Used)
		GPU[i].GrpcGPUfree = int64(memory.Free)
		GPU[i].GrpcGPUUUID = uuid
		GPU[i].GrpcGPUName = dname
		GPU[i].GrpcGPUIndex = i
		GPU[i].GrpcGPUtotal = int64(memory.Total)
		GPU[i].GrpcGPUtemp.Current = int(temperature)
		GPU[i].GrpcGPUpower = int(Power)
		GPU[i].GPURX = int(PCIRX)
		GPU[i].GPUTX = int(PCITX)
		GPU[i].GrpcGPUmpscount = podnum
		GPU[i].GrpcGPUtotalpower = int(power)
		GPU[i].GrpcGPUutil = int(util.Gpu)
		GPU[i].FanSpeed = int(fanspeed)
		bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database:  "metric",
			Precision: "s",
		})

		tags := map[string]string{"UUID": uuid} //8
		fields := map[string]interface{}{
			"Index":         i,                   //3
			"GPUName":       GPU[i].GrpcGPUName,  //2
			"memory(total)": GPU[i].GrpcGPUtotal, //10
			"memory(free)":  GPU[i].GrpcGPUfree,  //9
			"memory(used)":  GPU[i].GrpcGPUused,  //11
			"utilization":   GPU[i].GrpcGPUutil,  //13
			"Power":         GPU[i].GrpcGPUpower, //7
			"temperature":   GPU[i].GrpcGPUtemp,  //12
			"Podnum":        len(GPU[i].GPUPod),  //6
			"PciRx":         GPU[i].GPURX,        //4
			"PciTx":         GPU[i].GPUTX,        //5
			"FanSpeed":      GPU[i].FanSpeed,     //1
		}
		pt, err := influxdb.NewPoint("gpumetric", tags, fields, time.Now())
		if err != nil {
			fmt.Println("Error new point:", err.Error())
		}
		bp.AddPoint(pt)
		err = c.Write(bp)
		if err != nil {
			fmt.Println("Error write :", err.Error())
		}
		// fmt.Println("--------------Quantitative Measurement GPU Metric Collector Log--------------")
		// fmt.Println("Node Name : ", nodename)
		// fmt.Println("GPU UUID : ", uuid)
		// fmt.Println("GPU Name : ", string(*dname))
		// fmt.Println("GPU Index : ", i)
		// fmt.Println()
		// fmt.Println("Quantitative Measurement 7-1")
		// fmt.Println("GPU Utilization : ", *status.Utilization.GPU)
		// fmt.Println()
		// fmt.Println("Quantitative Measurement 7-2")
		// fmt.Println("GPU Memory (Total) : ", *memory, "MiB")
		// fmt.Println("GPU Memory (Free) : ", *Memory.Global.Free, "MiB")
		// fmt.Println("GPU Memory (Used) : ", *Memory.Global.Used, "MiB")
		// fmt.Println()
		// fmt.Println("Quantitative Measurement 7-3")
		// fmt.Println("GPU Temperature : ", *temperature, "°C")
		// fmt.Println()
		// fmt.Println("Quantitative Measurement 7-4")
		// fmt.Println("GPU Power (Total) : ", *power, "W")
		// fmt.Println("GPU Power (Used) : ", *Power, "W")
		//fmt.Println()
		//fmt.Printf("|%7v|  |%13v|  |%7v| |%7v| |%7v| MiB |%15v| C |%5v| |%5v| W \n", i, *status.Utilization.GPU, *memory, *Memory.Global.Free, *Memory.Global.Used, *temperature, *power, *Power)
	}
	go analyzer.Analyzer(*Node, c)

	bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  "metric",
		Precision: "s",
	})
	Node.TotalPodnum = 0
	for i := 0; i < len(Node.NodeGPU); i++ {
		Node.TotalPodnum += len(Node.NodeGPU[i].GPUPod)
	}
	tags := map[string]string{}
	fields := map[string]interface{}{
		"NodeCPU":       nodecpu,              //2
		"NodeMemory":    nodememory,           //3
		"uuid":          gpuuuid,              //9
		"Count":         count,                //1
		"NodeName":      nodename,             //4
		"NodeStorage":   Node.GrpcNodeStorage, //7
		"NodeNetworkRX": Node.NodeNetworkRX,   //6
		"NodeNetworkTX": Node.NodeNetworkTX,   //5
		"totalpod":      Node.TotalPodnum,     //8
	}
	pt, err := influxdb.NewPoint("multimetric", tags, fields, time.Now())
	if err != nil {
		fmt.Println("Error:", err.Error())
	}
	bp.AddPoint(pt)
	err = c.Write(bp)
	if err != nil {
		fmt.Println("Error:", err.Error())
	}
	// fmt.Println("--------------Multi Metric--------------")
	// fmt.Println("Node Name : ", nodename)
	// fmt.Println("Node CPU (Nano) : ", nodecpu)
	// fmt.Println("Node Memory (Bytes) : ", nodememory)
	// fmt.Println("GPU UUID : ", gpuuuid)
	// fmt.Println("GPU Count : ", count)
}
