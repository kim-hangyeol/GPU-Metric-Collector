package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	//"log"
	//"os/exec"

	"context"
	"metric-collector/customMetrics"
	"metric-collector/kubeletClient"
	"metric-collector/scrap"
	"metric-collector/storage"
	"os"
	"time"

	"metric-collector/grpcs"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"flag"
	"metric-collector/gpumetric"

	"github.com/NVIDIA/go-nvml/pkg/nvml"

	//"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
	_ "github.com/influxdata/influxdb1-client"
	influxdb "github.com/influxdata/influxdb1-client/v2"

	//"metric-collector/metricfactory"
	//"metric-collector/nvmemetriccollector"
	userpb "metric-collector/protos"

	"google.golang.org/grpc"
)

var collect_time = flag.Int("collecttime", 10, "Metric Collect Time")
var MaxLanes = 16

const portNumber = "9000"

type UserServer struct {
	userpb.UserServer
}

var Node []*grpcs.GrpcNode

var GPU []*grpcs.GrpcGPU

var gpuuuid []string

var prevNodeStorage int64

func main() {
	Node = append(Node, &grpcs.GrpcNode{})
	count := 0
	flag.Parse()
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		//log.Fatalf("Unable to initialize NVML: %v", ret)
		fmt.Printf("Unable to initialize NVML: %v\n", ret)
	} else {
		count, _ = nvml.DeviceGetCount()
		defer func() {
			ret := nvml.Shutdown()
			if ret != nvml.SUCCESS {
				//log.Fatalf("Unable to shutdown NVML: %v", ret)
				fmt.Printf("Unable to shutdown NVML: %v\n", ret)
			}
		}()

	}

	for i := 0; i < count; i++ {
		device, _ := nvml.DeviceGetHandleByIndex(i)
		uuid, _ := device.GetUUID()
		gpuuuid = append(gpuuuid, uuid)
		GPU = append(GPU, &grpcs.GrpcGPU{})
		maxclock, _ := device.GetMaxClockInfo(0)

		cudacore, _ := device.GetNumGpuCores()

		gpuflops := maxclock * uint32(cudacore) * 2 / 1000

		gpuarch, _ := device.GetArchitecture()

		GPU[i].GrpcGPUflops = int(gpuflops)
		GPU[i].GrpcGPUarch = int(gpuarch)
		gputemp, _ := device.GetTemperatureThreshold(3)
		GPU[i].GrpcGPUtemp.MaxOperating = int(gputemp)
		gputemp, _ = device.GetTemperatureThreshold(0)
		GPU[i].GrpcGPUtemp.Shutdown = int(gputemp)
		gputemp, _ = device.GetTemperatureThreshold(1)
		GPU[i].GrpcGPUtemp.Threshold = int(gputemp)
		// fmt.Printf("GPU %d FLOPS : %d\n", i, gpuflops)
		// fmt.Printf("  CudaCore : %d\n", cudacore)
		// fmt.Printf("  MaxGPUClock : %d\n", maxclock)
	}
	GetNvlinkInfo(Node[0])
	Node[0].NodeGPU = GPU
	Node[0].GrpcNodeUUID = gpuuuid
	Node[0].GrpcNodeCount = int(count)
	//gpumap := "gpumap"
	ip := "influxdb.gpu.svc.cluster.local"
	//ip := "10.244.2.2"
	port := "8086"
	url := "http://" + ip + ":" + port
	c, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr: url,
	})
	if err != nil {
		fmt.Println("Error creatring influx", err.Error())
	}
	dropdatabase := influxdb.Query{
		Command:  "drop database metric",
		Database: "_internal",
	}
	c.Query(dropdatabase)
	makedatabase := influxdb.Query{
		Command:  "create database metric",
		Database: "_internal",
	}
	name := os.Getenv("MY_NODE_NAME")
	Node[0].GrpcNodeName = name
	//fmt.Println(Node[0].GrpcNodeName)
	c.Query(makedatabase)
	defer c.Close()
	go grpcrun()
	// go analyzer()
	//fmt.Printf("nodename: %v\n", name)
	//cpu, err := exec.Command("grep", "-c", "processor", "/proc/cpuinfo").Output()
	//if err != nil {
	//	log.Fatalf("cpu exec error %v", err)
	//}
	//fmt.Printf("Total CPU Core : %v", string(cpu))
	//memory, err := exec.Command("grep", "MemTotal", "/proc/meminfo").Output()
	//if err != nil {
	//	log.Fatalf("mem exec error %v", err)
	//}
	//fmt.Printf(string(memory))
	// cleanup, err := dcgm.Init(dcgm.Embedded)
	// if err != nil {
	// 	log.Panicln(err)
	// }
	// defer cleanup()

	// // Request DCGM to start recording stats for GPU process fields
	// group, err := dcgm.WatchPidFields()
	// if err != nil {
	// 	log.Panicln(err)
	// }

	// Before retrieving process stats, wait few seconds for watches to be enabled and collect data
	//time.Sleep(3000 * time.Millisecond)

	for {
		//workdata := GetWorkData()
		data, totalcpu, nanocpu, totalmemory, nodememoey, nodetotalstorage, nodestorage := MemberMetricCollector()
		// if data == nil {
		// 	continue
		// }
		Node[0].GrpcNodeCPU = nanocpu
		Node[0].GrpcNodeMemory = nodememoey
		Node[0].GrpcNodetotalCPU = totalcpu
		Node[0].GrpcNodeTotalMemory = totalmemory
		Node[0].GrpcNodeTotalStorage = nodetotalstorage
		Node[0].GrpcNodeStorage = nodestorage
		Node[0].NodeNetworkRX = data.Metricsbatchs[0].Node.NetworkRxBytes.Value()
		Node[0].NodeNetworkTX = data.Metricsbatchs[0].Node.NetworkTxBytes.Value()
		go gpumetric.Gpumetric(c, nanocpu, nodememoey, name, GPU, data, Node[0])

		//nvmemetriccollector.Nvmemetric(c)
		//metricfactory.Factory(c)
		time.Sleep(time.Duration(*collect_time) * time.Second)
		// fmt.Println(*collect_time)
	}

}

// func GetWorkData() string {
// 	conn, err := grpc.Dial("10.0.5.24:30036", grpc.WithInsecure(), grpc.WithBlock())
// 	if err != nil {
// 		log.Fatalf("did not connect: %v", err)
// 	}
// 	defer conn.Close()
// 	c := userpb.NewUserClient(conn)

// 	// Contact the server and print out its response.
// 	//name := "gpu"
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
// 	var r *userpb.GetWorkNameResponse // 여기 수정중
// 	r, err = c.GetWorkName(ctx, &userpb.GetWorkNameRequest{Nodename: os.Getenv("MY_NODE_NAME")})
// 	if err != nil {
// 		log.Fatalf("could not greet: %v", err)
// 	}
// 	var workdata = r.GetWorkname()
// 	cancel()
// 	return workdata
// }

func MemberMetricCollector() (*storage.Collection, int64, int64, int64, int64, int64, int64) {
	//SERVER_IP := os.Getenv("GRPC_SERVER")
	//SERVER_PORT := os.Getenv("GRPC_PORT")
	//fmt.Println("ClusterMetricCollector Start")
	//grpcClient := protobuf.NewGrpcClient(SERVER_IP, SERVER_PORT)

	var period_int64 int64 = 5
	//var latencyTime float64 = 0

	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)
	//Node_list, _ := host_kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	MY_NODENAME := os.Getenv("MY_NODE_NAME")
	node, _ := host_kubeClient.CoreV1().Nodes().Get(context.TODO(), MY_NODENAME, metav1.GetOptions{})
	nodes := v1.NodeList{}
	nodes.Items = append(nodes.Items, *node)

	//fmt.Println("Get Metric Data From Kubelet")
	kubeletClient, _ := kubeletClient.NewKubeletClient()

	data, errs := scrap.Scrap(host_config, kubeletClient, nodes.Items)
	//fmt.Println(data)

	if errs != nil {
		fmt.Println(errs)
		time.Sleep(time.Duration(period_int64) * time.Second)
		//return nil, 0, 0
	}
	//fmt.Println("Convert Metric Data For gRPC")

	//latencyTime_string := fmt.Sprintf("%f", latencyTime)

	//grpc_data := convert(data, latencyTime_string)

	//fmt.Println("[gRPC Start] Send Metric Data")

	//rTime_start := time.Now()
	//r, err := grpcClient.SendMetrics(context.TODO(), grpc_data)
	//rTime_end := time.Since(rTime_start)

	//latencyTime = rTime_end.Seconds() - r.ProcessingTime

	//if err != nil {
	//fmt.Println("check")
	//	fmt.Println("could not connect : ", err)
	//	time.Sleep(time.Duration(period_int64) * time.Second)
	//fmt.Println("check2")
	//	continue
	//}
	//fmt.Println("[gRPC End] Send Metric Data")

	//period_int64 := r.Tick
	// _ = data

	//fmt.Println("[http Start] Post Metric Data to Custom Metric Server")
	token := host_config.BearerToken
	host := host_config.Host
	//client := host_kubeClient
	//fmt.Println("host: ", host)
	//fmt.Println("token: ", token)
	//fmt.Println("client: ", client)

	totalcpu, nodecpu, totalmemory, nodememory, nodetotalstorage, nodestorage := customMetrics.AddToPodCustomMetricServer(data, token, host)
	// fmt.Println(nodememory)
	return data, totalcpu, nodecpu, totalmemory, nodememory, nodetotalstorage, nodestorage
	//customMetrics.AddToDeployCustomMetricServer(data, token, host, client)
	//fmt.Println("[http End] Post Metric Data to Custom Metric Server")

	//period_int64 = r.Tick

	//if period_int64 > 0 && err == nil {

	//fmt.Println("period : ",time.Duration(period_int64))
	//	fmt.Println("Wait ", time.Duration(period_int64)*time.Second, "...")
	//	time.Sleep(time.Duration(period_int64) * time.Second)
	//} else {
	//	fmt.Println("--- Fail to get period")
	//	time.Sleep(5 * time.Second)
	//}
}

func (s *UserServer) GetNode(ctx context.Context, req *userpb.GetNodeRequest) (*userpb.GetNodeResponse, error) {
	var userNodeMessage *userpb.NodeMessage
	// var maxgpumemory = int64(0)
	// for i := range gpuuuid {
	// 	if maxgpumemory < int64(GPU[i].GrpcGPUtotal) {
	// 		maxgpumemory = int64(GPU[i].GrpcGPUtotal)
	// 	}

	// }
	var nodedata = &userpb.NodeMessage{
		NodeName: Node[0].GrpcNodeName,
		// NodeTotalcpu:     Node[0].GrpcNodetotalCPU,
		NodeCpu: Node[0].GrpcNodeCPU,
		// GpuCount:         int64(Node[0].GrpcNodeCount),
		// NodeTotalmemory:  Node[0].GrpcNodeTotalMemory,
		NodeMemory: Node[0].GrpcNodeMemory,
		// NodeTotalstorage: Node[0].GrpcNodeTotalStorage,
		NodeStorage: Node[0].GrpcNodeStorage,
		// GpuUuid:          strings.Join(Node[0].GrpcNodeUUID, " "),
		// MaxGpuMemory:     maxgpumemory,
	}
	if prevNodeStorage != Node[0].GrpcNodeTotalStorage {
		nodedata.NodeTotalstorage = Node[0].GrpcNodeTotalStorage
	}
	prevNodeStorage = Node[0].GrpcNodeTotalStorage
	userNodeMessage = nodedata
	//fmt.Println(nodedata)
	return &userpb.GetNodeResponse{
		NodeMessage: userNodeMessage,
	}, nil
}

// ListUsers returns all user messages
func (s *UserServer) GetGPU(ctx context.Context, req *userpb.GetGPURequest) (*userpb.GetGPUResponse, error) {
	var userGPUMessage *userpb.GPUMessage
	uuid := req.GpuUuid

	for i, j := range gpuuuid {
		if j != uuid {
			continue
		}
		if GPU[i].GrpcGPUmpscount < 0 {
			GPU[i].GrpcGPUmpscount = 0
		}
		var gpudata = &userpb.GPUMessage{
			// GpuUuid:   GPU[i].GrpcGPUUUID,
			GpuUsed: uint64(GPU[i].GrpcGPUused),
			// GpuName:   GPU[i].GrpcGPUName,
			// GpuIndex:  int64(GPU[i].GrpcGPUIndex),
			// GpuTotal:  uint64(GPU[i].GrpcGPUtotal),
			GpuFree:  uint64(GPU[i].GrpcGPUfree),
			GpuTemp:  int64(GPU[i].GrpcGPUtemp.Current),
			GpuPower: int64(GPU[i].GrpcGPUpower),
			MpsCount: int64(GPU[i].GrpcGPUmpscount),
			GpuUtil:  int64(GPU[i].GrpcGPUutil),
			// GpuFlops:  int64(GPU[i].GrpcGPUflops),
			// GpuArch:   int64(GPU[i].GrpcGPUarch),
			// GpuTpower: int64(GPU[i].GrpcGPUtotalpower),
		}
		userGPUMessage = gpudata
		break
	}
	// fmt.Println(userGPUMessage)
	//fmt.Println(userGPUMessage)
	return &userpb.GetGPUResponse{
		GpuMessage: userGPUMessage,
	}, nil
}

func (s *UserServer) GetInitData(ctx context.Context, req *userpb.InitRequest) (*userpb.InitMessage, error) {

	time.Sleep(time.Second * time.Duration(*collect_time))
	var maxgpumemory = int64(0)
	for i := 0; i < len(gpuuuid); i++ {
		if maxgpumemory < int64(GPU[i].GrpcGPUtotal) {
			maxgpumemory = int64(GPU[i].GrpcGPUtotal)
		}

	}

	var gpu1index []string
	var gpu2index []string
	var lanes []int32

	var grpcgpudata []*userpb.GPUMessage

	for i := 0; i < len(Node[0].NvLinkInfo); i++ {
		gpu1index = append(gpu1index, Node[0].NvLinkInfo[i].GPU1UUID)
		gpu2index = append(gpu2index, Node[0].NvLinkInfo[i].GPU2UUID)
		lanes = append(lanes, int32(Node[0].NvLinkInfo[i].CountLink))
	}

	var Nodedata = &userpb.NodeMessage{
		NodeName:         Node[0].GrpcNodeName,
		NodeTotalcpu:     Node[0].GrpcNodetotalCPU,
		NodeCpu:          Node[0].GrpcNodeCPU,
		GpuCount:         int64(Node[0].GrpcNodeCount),
		NodeTotalmemory:  Node[0].GrpcNodeTotalMemory,
		NodeMemory:       Node[0].GrpcNodeMemory,
		NodeTotalstorage: Node[0].GrpcNodeTotalStorage,
		NodeStorage:      Node[0].GrpcNodeStorage,
		GpuUuid:          strings.Join(Node[0].GrpcNodeUUID, " "),
		MaxGpuMemory:     maxgpumemory,
		Gpu1Index:        gpu1index,
		Gpu2Index:        gpu2index,
		Lanecount:        lanes,
	}
	// userInitMessage.InitNode.NodeName = Node[0].GrpcNodeName
	// userInitMessage.InitNode.NodeTotalcpu = Node[0].GrpcNodetotalCPU
	// userInitMessage.InitNode.NodeCpu = Node[0].GrpcNodeCPU
	// userInitMessage.InitNode.GpuCount = int64(Node[0].GrpcNodeCount)
	// userInitMessage.InitNode.NodeTotalmemory = Node[0].GrpcNodeTotalMemory
	// userInitMessage.InitNode.NodeMemory = Node[0].GrpcNodeMemory
	// userInitMessage.InitNode.NodeTotalstorage = Node[0].GrpcNodeTotalStorage
	// userInitMessage.InitNode.NodeStorage = Node[0].GrpcNodeStorage
	// userInitMessage.InitNode.GpuUuid = strings.Join(Node[0].GrpcNodeUUID, " ")
	// userInitMessage.InitNode.MaxGpuMemory = maxgpumemory

	prevNodeStorage = Node[0].GrpcNodeTotalStorage
	// userInitMessage.InitNode.Gpu1Index = Node[0].nv

	for i := 0; i < len(gpuuuid); i++ {
		// userInitMessage.InitGPU = append(userInitMessage.InitGPU, &userpb.GPUMessage{})
		// userInitMessage.InitGPU[i].GpuUuid = j
		if GPU[i].GrpcGPUmpscount < 0 {
			GPU[i].GrpcGPUmpscount = 0
		}

		var gpudata = &userpb.GPUMessage{
			GpuUuid:   GPU[i].GrpcGPUUUID,
			GpuUsed:   uint64(GPU[i].GrpcGPUused),
			GpuName:   GPU[i].GrpcGPUName,
			GpuIndex:  int64(GPU[i].GrpcGPUIndex),
			GpuTotal:  uint64(GPU[i].GrpcGPUtotal),
			GpuFree:   uint64(GPU[i].GrpcGPUfree),
			GpuTemp:   int64(GPU[i].GrpcGPUtemp.Current),
			GpuPower:  int64(GPU[i].GrpcGPUpower),
			MpsCount:  int64(GPU[i].GrpcGPUmpscount),
			GpuUtil:   int64(GPU[i].GrpcGPUutil),
			GpuFlops:  int64(GPU[i].GrpcGPUflops),
			GpuArch:   int64(GPU[i].GrpcGPUarch),
			GpuTpower: int64(GPU[i].GrpcGPUtotalpower),
		}
		// userInitMessage.InitGPU[i].GpuUsed = uint64(GPU[i].GrpcGPUused)
		// userInitMessage.InitGPU[i].GpuName = GPU[i].GrpcGPUName
		// userInitMessage.InitGPU[i].GpuIndex = int64(GPU[i].GrpcGPUIndex)
		// userInitMessage.InitGPU[i].GpuTotal = uint64(GPU[i].GrpcGPUtotal)
		// userInitMessage.InitGPU[i].GpuFree = uint64(GPU[i].GrpcGPUfree)
		// userInitMessage.InitGPU[i].GpuTemp = int64(GPU[i].GrpcGPUtemp)
		// userInitMessage.InitGPU[i].GpuPower = int64(GPU[i].GrpcGPUpower)
		// userInitMessage.InitGPU[i].MpsCount = int64(GPU[i].GrpcGPUmpscount)
		// userInitMessage.InitGPU[i].GpuFlops = int64(GPU[i].GrpcGPUflops)
		// userInitMessage.InitGPU[i].GpuArch = int64(GPU[i].GrpcGPUarch)
		// userInitMessage.InitGPU[i].GpuTpower = int64(GPU[i].GrpcGPUtotalpower)
		grpcgpudata = append(grpcgpudata, gpudata)
	}

	// fmt.Println(userGPUMessage)
	//fmt.Println(userGPUMessage)
	return &userpb.InitMessage{
		InitNode: Nodedata,
		InitGPU:  grpcgpudata,
	}, nil
}

func grpcrun() {
	lis, err := net.Listen("tcp", ":"+portNumber)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	userpb.RegisterUserServer(grpcServer, &UserServer{})

	log.Printf("start gRPC server on %s port", portNumber)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}

func GetNvlinkInfo(node *grpcs.GrpcNode) {
	count, err := nvml.DeviceGetCount()
	if err != nvml.SUCCESS {
		return
	}
	var nvlinkStatus []storage.NvlinkStatus
	for i := 0; i < count; i++ {
		nvlinkStatus = append(nvlinkStatus, storage.NvlinkStatus{})
		device, _ := nvml.DeviceGetHandleByIndex(i)
		pciinfo, _ := device.GetPciInfo()
		tmparray := pciinfo.BusId
		var bytebus [32]byte
		for j := 0; j < 32; j++ {
			bytebus[j] = byte(tmparray[i])
		}
		nvlinkStatus[i].BusID = string(bytebus[:])
		nvlinkStatus[i].UUID, _ = device.GetUUID()
		nvlinkStatus[i].Lanes = make(map[string]int)
		// nvlinkStatus[i].Lanes
		for j := 0; j < MaxLanes; j++ {
			P2PPciInfo, err := device.GetNvLinkRemotePciInfo(j)
			if err != nvml.SUCCESS {
				break
			}
			tmparray := P2PPciInfo.BusId
			var bytebus [32]byte
			for k := 0; k < 32; k++ {
				bytebus[k] = byte(tmparray[k])
			}
			val, exists := nvlinkStatus[i].Lanes[string(bytebus[:])]
			if !exists {
				P2PDevice, err := nvml.DeviceGetHandleByPciBusId(string(bytebus[:]))
				if err != nvml.SUCCESS {
					fmt.Println("error can get device handle")
				} else {
					P2PIndex, _ := P2PDevice.GetIndex()
					if P2PIndex > j {

						// if P2PDevice.GetIndex()
						types, _ := device.GetNvLinkRemoteDeviceType(j)
						nvlinkStatus[i].Lanes[string(bytebus[:])] = 1
						nvlinkStatus[i].P2PDeviceType = append(nvlinkStatus[i].P2PDeviceType, int(types))
						P2PUUID, _ := P2PDevice.GetUUID()
						nvlinkStatus[i].P2PUUID = append(nvlinkStatus[i].P2PUUID, P2PUUID)
						nvlinkStatus[i].P2PBusID = append(nvlinkStatus[i].P2PBusID, string(bytebus[:]))
					}
				}
				// if j == 0 {
				// 	nvlinkStatus[i].Lanes = make(map[string]int)
				// }

			} else {
				nvlinkStatus[i].Lanes[string(bytebus[:])] = val + 1
			}
		}
		for j := 0; j < len(nvlinkStatus[i].P2PUUID); j++ {
			var tmpnv grpcs.NvLink
			tmpnv.GPU1UUID = nvlinkStatus[i].UUID
			tmpnv.GPU2UUID = nvlinkStatus[i].P2PUUID[j]
			tmpnv.CountLink = nvlinkStatus[i].Lanes[nvlinkStatus[i].P2PBusID[j]]
			node.NvLinkInfo = append(node.NvLinkInfo, tmpnv)
		}
	}
	// fmt.Println(node.NvLinkInfo)

}
