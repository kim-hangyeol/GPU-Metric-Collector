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

const portNumber = "9000"

type UserServer struct {
	userpb.UserServer
}

var Node []*grpcs.GrpcNode

var GPU []*grpcs.GrpcGPU

var Podmap []*storage.GPUMAP

var gpuuuid []string

func main() {
	count := 0
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
	}
	Node = append(Node, &grpcs.GrpcNode{})
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
		data, totalcpu, nanocpu, totalmemory, nodememoey, nodetotalstorage, nodestorage := MemberMetricCollector()
		// if data == nil {
		// 	continue
		// }
		gpumetric.Gpumetric(c, nanocpu, nodememoey, name, GPU, data, Podmap)
		Node[0].GrpcNodeCPU = nanocpu
		Node[0].GrpcNodeMemory = nodememoey
		Node[0].GrpcNodetotalCPU = totalcpu
		Node[0].GrpcNodeTotalMemory = totalmemory
		Node[0].GrpcNodeTotalStorage = nodetotalstorage
		Node[0].GrpcNodeStorage = nodestorage
		//nvmemetriccollector.Nvmemetric(c)
		//metricfactory.Factory(c)

		time.Sleep(10 * time.Second)
	}

}

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
	var maxgpumemory = int64(0)
	for i := range gpuuuid {
		if maxgpumemory < int64(GPU[i].GrpcGPUtotal) {
			maxgpumemory = int64(GPU[i].GrpcGPUtotal)
		}

	}
	var nodedata = &userpb.NodeMessage{
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
	}
	userNodeMessage = nodedata
	// fmt.Println(nodedata)
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
			GpuUuid:  GPU[i].GrpcGPUUUID,
			GpuUsed:  GPU[i].GrpcGPUused,
			GpuName:  GPU[i].GrpcGPUName,
			GpuIndex: int64(GPU[i].GrpcGPUIndex),
			GpuTotal: GPU[i].GrpcGPUtotal,
			GpuFree:  GPU[i].GrpcGPUfree,
			GpuTemp:  int64(GPU[i].GrpcGPUtemp),
			GpuPower: int64(GPU[i].GrpcGPUpower),
			MpsCount: int64(GPU[i].GrpcGPUmpscount),
		}
		userGPUMessage = gpudata
		break
	}
	// fmt.Println(userGPUMessage)

	return &userpb.GetGPUResponse{
		GpuMessage: userGPUMessage,
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
