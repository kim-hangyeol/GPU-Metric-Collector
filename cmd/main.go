package main

import (
	"fmt"
	//"log"
	//"os/exec"

	"context"
	"metric-collector/customMetrics"
	"metric-collector/kubeletClient"
	"metric-collector/scrap"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	//"github.com/NVIDIA/go-nvml/pkg/nvml"
	"metric-collector/gpumetric"

	_ "github.com/influxdata/influxdb1-client"
	influxdb "github.com/influxdata/influxdb1-client/v2"
	//"metric-collector/metricfactory"
	//"metric-collector/nvmemetriccollector"
)

func main() {
	//gpumap := "gpumap"
	ip := "influxdb.gpu.svc.cluster.local"
	//ip := "10.102.31.195"
	port := "8086"
	url := "http://" + ip + ":" + port
	c, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr: url,
	})
	if err != nil {
		fmt.Println("Error creatring influx", err.Error())
	}
	defer c.Close()

	// makedatabase := influxdb.Query{
	// 	Command:  "create database gpumap",
	// 	Database: gpumap,
	// }
	// c.Query(makedatabase)
	name := os.Getenv("MY_NODE_NAME")

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
	nanocpu := ""
	nodememoey := ""
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
		//nanocpu, nodememoey = MemberMetricCollector()
		gpumetric.Gpumetric(c, nanocpu, nodememoey, name)
		//nvmemetriccollector.Nvmemetric(c)
		//metricfactory.Factory(c)

		time.Sleep(1000000000)
	}

}

func MemberMetricCollector() (string, string) {
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
		return "", ""
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

	nodecpu, nodememory := customMetrics.AddToPodCustomMetricServer(data, token, host)
	return nodecpu, nodememory
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
