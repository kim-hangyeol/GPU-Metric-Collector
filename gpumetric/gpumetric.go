package gpumetric

import (
	"fmt"
	"log"
	"metric-collector/gpumpsmetric"
	"metric-collector/grpcs"
	"metric-collector/storage"
	"strconv"
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

func Gpumetric(c influxdb.Client, nodecpu int64, nodememory int64, nodename string, GPU []*grpcs.GrpcGPU, data *storage.Collection, podmap []*storage.GPUMAP) {
	//func Gpumetric(c influxdb.Client, nvml *nvml1) {
	//name := os.Getenv("MY_NODE_NAME")

	//fmt.Printf("nodename: %v\n", name)
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
		podnum := gpumpsmetric.Gpumpsmetric(device, int(count), c, data, podmap[i].PodMetric)

		//process := status.Processes
		//fmt.Printf("%v \n",process)

		//uuid := device.UUID //uuid
		//fmt.Printf("GPU UUID : %v\n", uuid)

		dname, _ := device.GetName() //gpuname
		//fmt.Printf("%v\n", *dname)
		//fmt.Printf("GPU 이름 : %v\n", *dname)

		memory, _ := device.GetMemoryInfo() //메모리 정보 total
		//fmt.Printf("%v\n", *memory)
		//fmt.Printf("메모리 총량(MiB) : %v\n", *memory)

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
		//fmt.Printf("%v\n", *power)
		//fmt.Printf("파워 총량(W) : %v\n", *power)

		Power, _ := device.GetPowerUsage() // 파워 사용량
		//fmt.Printf("%v\n", *Power)
		//fmt.Printf("파워 사용량(W) : %v\n", *Power)

		temperature, _ := device.GetTemperature(0) //온도
		//fmt.Printf("%v\n", *temperature)
		//fmt.Printf("현재 온도(°C) : %v\n", *temperature)

		/*pci := status.PCI
		fmt.Printf("bar1 memory 사용량 : %v\n", pci.BAR1Used)

		fmt.Printf("RX, TX 사용량 : %v %v\n", pci.Throughput.RX, pci.Throughput.TX)*/

		//pci := device.PCI //pci
		//fmt.Printf("PCI (BusID, BAR1 Memory, BandWidth): %v %v %v\n", pci.BusID, *(pci.BAR1), *(pci.Bandwidth))
		gpuuuid = append(gpuuuid, uuid)

		GPU[i].GrpcGPUused = memory.Used
		GPU[i].GrpcGPUfree = memory.Free
		GPU[i].GrpcGPUUUID = uuid
		GPU[i].GrpcGPUName = dname
		GPU[i].GrpcGPUIndex = i
		GPU[i].GrpcGPUtotal = memory.Total
		GPU[i].GrpcGPUtemp = int(temperature)
		GPU[i].GrpcGPUpower = int(Power)
		GPU[i].GrpcGPUmpscount = podnum
		GPU[i].GrpcGPUtotalpower = int(power)

		bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database:  "metric",
			Precision: "s",
		})

		tags := map[string]string{"UUID": uuid}
		fields := map[string]interface{}{
			"Index":         i,
			"GPUName":       string(dname),
			"memory(total)": strconv.FormatUint(memory.Total, 10),
			"memory(free)":  strconv.FormatUint(memory.Free, 10),
			"memory(used)":  strconv.FormatUint(memory.Used, 10),
			"Power":         strconv.FormatUint(uint64(Power), 10),
			"temperature":   strconv.FormatUint(uint64(temperature), 10),
		}
		pt, err := influxdb.NewPoint("gpumetric", tags, fields, time.Now())
		if err != nil {
			fmt.Println("Error:", err.Error())
		}
		bp.AddPoint(pt)
		err = c.Write(bp)
		if err != nil {
			fmt.Println("Error:", err.Error())
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
		fmt.Println()
		//fmt.Printf("|%7v|  |%13v|  |%7v| |%7v| |%7v| MiB |%15v| C |%5v| |%5v| W \n", i, *status.Utilization.GPU, *memory, *Memory.Global.Free, *Memory.Global.Used, *temperature, *power, *Power)
	}

	bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  "metric",
		Precision: "s",
	})

	tags := map[string]string{}
	fields := map[string]interface{}{
		"NodeCPU":    nodecpu,
		"NodeMemory": nodememory,
		"uuid":       gpuuuid,
		"Count":      count,
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
	// fmt.Println("--------------Multi Metric--------------")
	// fmt.Println("Node Name : ", nodename)
	// fmt.Println("Node CPU (Nano) : ", nodecpu)
	// fmt.Println("Node Memory (Bytes) : ", nodememory)
	// fmt.Println("GPU UUID : ", gpuuuid)
	// fmt.Println("GPU Count : ", count)
}
