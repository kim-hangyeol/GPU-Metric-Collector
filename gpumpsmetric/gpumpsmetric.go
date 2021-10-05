package gpumpsmetric

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	_ "github.com/influxdata/influxdb1-client"
	influxdb "github.com/influxdata/influxdb1-client/v2"
)

type Runningpod struct {
	Data struct {
		PodDeviceEntries []struct {
			PodUID        string `json:"PodUID"`
			ContainerName string `json:"ContainerName"`
			ResourceName  string `json:"ResourceName"`
			DeviceIDs     struct {
				Num0 []string `json:"0"`
			} `json:"DeviceIDs"`
			AllocResp string `json:"AllocResp"`
		} `json:"PodDeviceEntries"`
		RegisteredDevices struct {
			K8SKetimpsgpu []string `json:"keti.com/mpsgpu"`
		} `json:"RegisteredDevices"`
	} `json:"Data"`
	Checksum int `json:"Checksum"`
}

type GpuMpsMap struct {
	UUID string
	// useMemory int
	Container string
	Pod       string
	PID       uint
	Index     int
	RunFlag   int
}

// const (
// 	processInfo = `----------------------------------------------------------------------
// GPU ID			     : {{.GPU}}
// ----------Execution Stats---------------------------------------------
// PID                          : {{.PID}}
// Name                         : {{or .Name "N/A"}}
// Start Time                   : {{.ProcessUtilization.StartTime.String}}
// End Time                     : {{.ProcessUtilization.EndTime.String}}
// ----------Performance Stats-------------------------------------------
// Energy Consumed (Joules)     : {{or .ProcessUtilization.EnergyConsumed "N/A"}}
// Max GPU Memory Used (bytes)  : {{or .Memory.GlobalUsed "N/A"}}
// Avg SM Clock (MHz)           : {{or .Clocks.Cores "N/A"}}
// Avg Memory Clock (MHz)       : {{or .Clocks.Memory "N/A"}}
// Avg SM Utilization (%)       : {{or .GpuUtilization.Memory "N/A"}}
// Avg Memory Utilization (%)   : {{or .GpuUtilization.GPU "N/A"}}
// Avg PCIe Rx Bandwidth (MB)   : {{or .PCI.Throughput.Rx "N/A"}}
// Avg PCIe Tx Bandwidth (MB)   : {{or .PCI.Throughput.Tx "N/A"}}
// ----------Event Stats-------------------------------------------------
// Single Bit ECC Errors        : {{or .Memory.ECCErrors.SingleBit "N/A"}}
// Double Bit ECC Errors        : {{or .Memory.ECCErrors.DoubleBit "N/A"}}
// Critical XID Errors          : {{.XIDErrors.NumErrors}}
// ----------Slowdown Stats----------------------------------------------
// Due to - Power (%)           : {{or .Violations.Power "N/A"}}
//        - Thermal (%)         : {{or .Violations.Thermal "N/A"}}
//        - Reliability (%)     : {{or .Violations.Reliability "N/A"}}
//        - Board Limit (%)     : {{or .Violations.BoardLimit "N/A"}}
//        - Low Utilization (%) : {{or .Violations.LowUtilization "N/A"}}
//        - Sync Boost (%)      : {{or .Violations.SyncBoost "N/A"}}
// ----------Process Utilization-----------------------------------------
// Avg SM Utilization (%)       : {{or .ProcessUtilization.SmUtil "N/A"}}
// Avg Memory Utilization (%)   : {{or .ProcessUtilization.MemUtil "N/A"}}
// ----------------------------------------------------------------------
// `
// )

/*type Device struct {
	UUID                   string
	GetAllRunningProcesses func() ([]ProcessInfo, error)
	// contains filtered or unexported fields
}

type ProcessInfo struct {
	PID        uint
	Name       string
	MemoryUsed uint64
	Type       ProcessType
}

type ProcessType uint

const (
	Compute ProcessType = iota
	Graphics
	ComputeAndGraphics
)*/

// func findpid(pid uint, pidtable []uint) int {
// 	for i := 0; i < len(pidtable); i++ {
// 		if pid == pidtable[i] {
// 			return 1
// 		}
// 	}
// 	return 0
// }

func getpodlist() (*v1.PodList, error) {
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)
	MY_NODENAME := os.Getenv("MY_NODE_NAME")
	selector := fields.SelectorFromSet(fields.Set{"spec.nodeName": MY_NODENAME, "status.phase": "Running"})
	podlist, err := host_kubeClient.CoreV1().Pods("userpod").List(context.TODO(), metav1.ListOptions{
		FieldSelector: selector.String(),
		LabelSelector: labels.Everything().String(),
	})
	for i := 0; i < 3 && err != nil; i++ {
		podlist, err = host_kubeClient.CoreV1().Pods("userpod").List(context.TODO(), metav1.ListOptions{
			FieldSelector: selector.String(),
			LabelSelector: labels.Everything().String(),
		})
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get Pods assigned to node %v", MY_NODENAME)
	}
	return podlist, nil
}

// var firstnum = 0

func Gpumpsmetric(device nvml.Device, count int, c influxdb.Client) {
	var gpumpsmap [10]GpuMpsMap
	//var pidtable []uint

	//var mapping [10]int
	UUID := device.UUID
	mps, err := device.GetAllRunningProcesses()
	//fmt.Println(mps)
	if err != nil {
		log.Fatalln(err)
	}
	loopnum := 0
	var runpodlist []v1.Pod
	var temppod v1.Pod
	var runpod string
	for {
		if loopnum != 0 {
			time.Sleep(100000000)
		}
		podnum := 0
		podlist, err := getpodlist()
		if err != nil {
			log.Fatalln(err)
		}
		for i := 0; i < len(podlist.Items); i++ { //container creating 때문에 갯수가 안맞는듯?
			annotation := podlist.Items[i].ObjectMeta.Annotations["UUID"]
			annotationUUID := strings.Split(annotation, ",")
			if len(annotationUUID) > 1 {
				for j := 0; j < len(annotationUUID); j++ {
					if annotationUUID[j] == UUID {
						gpumpsmap[podnum].Container = podlist.Items[i].Spec.Containers[0].Name
						gpumpsmap[podnum].Pod = podlist.Items[i].ObjectMeta.Name
						podnum++
					}
				}
			} else {
				if podlist.Items[i].ObjectMeta.Annotations["UUID"] == UUID {
					//fmt.Printf("running pod name : %v\n", podlist.Items[i].ObjectMeta.Name)
					gpumpsmap[podnum].Container = podlist.Items[i].Spec.Containers[0].Name
					gpumpsmap[podnum].Pod = podlist.Items[i].ObjectMeta.Name
					podnum++
				}
			}
		}
		if podnum == len(mps)-1 {
			runpodlist = append(runpodlist, podlist.Items...)
			//fmt.Printf("running pod num : %v\nrunning process num : %v\n", podnum, len(mps))
			//runpodlist = *podlist
			break
		}
		if loopnum == 20 {
			// fmt.Println("----------------------------------------------------")
			// fmt.Printf("running pod name: ")
			// for k := 0; k < len(podlist.Items); k++ {
			// 	fmt.Printf("%v ", podlist.Items[k].ObjectMeta.Name)
			// }
			// fmt.Println()
			break
		}
		loopnum++
	}
	for i := 0; i < len(runpodlist); i++ {
		for j := 0; j < len(runpodlist); j++ {
			if runpodlist[i].Status.StartTime.Before(runpodlist[j].Status.StartTime) {
				temppod = runpodlist[i]
				runpodlist[i] = runpodlist[j]
				runpodlist[j] = temppod
			}
		}
	}
	podnum := 0
	for i := 0; i < len(runpodlist); i++ { //container creating 때문에 갯수가 안맞는듯?
		annotation := runpodlist[i].ObjectMeta.Annotations["UUID"]
		annotationUUID := strings.Split(annotation, ",")
		if len(annotationUUID) > 1 {
			for j := 0; j < len(annotationUUID); j++ {
				if annotationUUID[j] == UUID {
					gpumpsmap[podnum].Container = runpodlist[i].Spec.Containers[0].Name
					gpumpsmap[podnum].Pod = runpodlist[i].ObjectMeta.Name
					runpod = runpod + runpodlist[i].ObjectMeta.Name
					podnum++
				}
			}
		} else {
			if runpodlist[i].ObjectMeta.Annotations["UUID"] == UUID {
				//fmt.Printf("running pod name : %v\n", podlist.Items[i].ObjectMeta.Name)
				gpumpsmap[podnum].Container = runpodlist[i].Spec.Containers[0].Name
				gpumpsmap[podnum].Pod = runpodlist[i].ObjectMeta.Name
				runpod = runpod + runpodlist[i].ObjectMeta.Name
				podnum++
			}
		}
	}

	//fmt.Println(mps)
	/*runcontainers, ret := ioutil.ReadFile("/var/lib/kubelet/device-plugins/kubelet_internal_checkpoint") //device-plugins
	//runcontainers, ret := ioutil.ReadFile("/root/workspace/kmc/gputest/checkpointtestsecond/kubelet_internal_checkpoint") //device-plugins
	if ret != nil {
		log.Fatalf("%v", ret)
	}
	var con Runningpod
	//fmt.Println(string(runcontainers))
	json.Unmarshal(runcontainers, &con)
	for i := 1; i < len(con.Data.PodDeviceEntries); i++ {
		if UUID == "GPU-f6db4146-092d-146f-0814-8ff90b04f3d2" && i%2 == 1 {
			gpumpsmap[i/2].Container = con.Data.PodDeviceEntries[i].ContainerName
		} else if UUID == "GPU-a06cd524-72c4-d6f0-4eda-d64af512dd8b" && i%2 == 0 {
			gpumpsmap[i/2-1].Container = con.Data.PodDeviceEntries[i].ContainerName
		}
	}*/
	//fmt.Println(gpumpsmap)
	//fmt.Println(len(con.Data.PodDeviceEntries))
	// fmt.Println("----------------------------------------------------")
	// fmt.Printf("running pod name: ")
	// for k := 0; k < len(runpodlist.Items); k++ {
	// 	fmt.Printf("%v ", runpodlist.Items[k].ObjectMeta.Name)
	// }
	//fmt.Println(mps)
	//fmt.Println()
	fmt.Println("GPU UUID : ", UUID)
	fmt.Println("GPU running process : ", mps)
	fmt.Println("GPU running pod : ", runpod)
	fmt.Println("----------------------------------------------------")
	fmt.Printf("    GPUMPSMAP ( GPU UUID : %v )\n", UUID)
	fmt.Printf("   MPS NUM      Used Memory        PodName                          Container Name\n") //프로세스의 이름이 안나오네
	for i := 1; i < len(mps); i++ {
		if mps[i].Name != "nvidia-cuda-mps-server" {
			fmt.Printf("MPS |%4v| : |%10vMiB| |%30v|  |%20v|\n", i-1, mps[i].MemoryUsed, gpumpsmap[i-1].Pod, gpumpsmap[i-1].Container)
		} else {
			fmt.Printf("MPS |%4v| : |%10vMiB| |%30v|  |%20v|\n", i-1, mps[i-1].MemoryUsed, gpumpsmap[i-1].Pod, gpumpsmap[i-1].Container)
		}
	}
	fmt.Println("----------------------------------------------------")

	// bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
	// 	Database:  "multimetric",
	// 	Precision: "s",
	// })

	// tags := map[string]string{"UUID": UUID}
	// fields := map[string]interface{}{
	// 	"NodeCPU":    nodecpu,
	// 	"NodeMemory": nodememory,
	// 	"uuid":       gpuuuid,
	// 	"Count":      count,
	// }
	// pt, err := influxdb.NewPoint("metric", tags, fields, time.Now())
	// if err != nil {
	// 	fmt.Println("Error:", err.Error())
	// }
	// bp.AddPoint(pt)
	// err = c.Write(bp)
	// if err != nil {
	// 	fmt.Println("Error:", err.Error())
	// }

	// t := template.Must(template.New("Process").Parse(processInfo))
	// for i := 0; i < len(mps); i++ {
	// 	pidInfo, err := dcgm.GetProcessInfo(group, mps[i].PID)
	// 	if err != nil {
	// 		log.Panicln(err)
	// 	}
	// 	if err = t.Execute(os.Stdout, pidInfo); err != nil {
	// 		log.Panicln("Template error:", err)
	// 	}
	// }

	// loc, err := time.LoadLocation("Asia/Seoul")
	// if err != nil {
	// 	log.Fatalf("time error : %v", err)
	// }
	// t := time.Now().In(loc)
	// fmt.Println("Success Time : ", t)

	/*for i := 0; i < len(mps); i++ {    이거쓸거임 나중에
		if mps[i].Name != "nvidia-cuda-mps-server" {
			gpumpsmap[i].PID = mps[i].PID
			gpumpsmap[i].useMemory = int(mps[i].MemoryUsed)
			gpumpsmap[i].Process = mps[i].Name
			gpumpsmap[i].UUID = UUID
		}
	}
	processnum := 0
	for i := 0; i < len(con.Data.PodDeviceEntries); i++ {
		for j := 0; j < len(con.Data.PodDeviceEntries[i].DeviceIDs.Num0); j++ {
			tmp := con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j][:len(con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j])-2]
			mpsnum, err := strconv.Atoi(con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j][len(con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j])-1 : len(con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j])])
			if err != nil {
				log.Fatalf("strconv error : %v\n", err)
			}
			if tmp == UUID &&
		}
	}*/

	/*for i := 0; i < len(con.Data.PodDeviceEntries); i++ { //파드 수    컨테이너가 생성되었을때 여기서 등록
		//fmt.Println(len(con.Data.PodDeviceEntries[i].DeviceIDs.Num0))
		for j := 0; j < len(con.Data.PodDeviceEntries[i].DeviceIDs.Num0); j++ { //파드에 할당된 gpu 수
			//tmp := con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j][:40]
			tmp := con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j][:len(con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j])-2]
			//mpsnum, err := strconv.Atoi(con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j][41:42])
			mpsnum, err := strconv.Atoi(con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j][len(con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j])-1 : len(con.Data.PodDeviceEntries[i].DeviceIDs.Num0[j])])
			if err != nil {
				log.Fatalf("%v\n", err)
			}
			//fmt.Println(tmp)
			if tmp == UUID && gpumpsmap[mpsnum].UUID == "" {
				//fmt.Println(con.Data.PodDeviceEntries[i].ContainerName)
				for k := 0; k < len(mps); k++ { //실행중인 프로세스와 매핑
					if mps[k].Name != "nvidia-cuda-mps-server" && findpid(mps[k].PID, pidtable) != 1 {
						pidtable = append(pidtable, mps[k].PID)
						gpumpsmap[mpsnum].UUID = UUID
						gpumpsmap[mpsnum].Container = con.Data.PodDeviceEntries[i].ContainerName
						gpumpsmap[mpsnum].useMemory = int(mps[k].MemoryUsed)
						gpumpsmap[mpsnum].PID = mps[k].PID
						gpumpsmap[mpsnum].Process = mps[k].Name
						gpumpsmap[mpsnum].Index = k
						break
					}
				}
			}
		}
	}
	for i := 0; i < len(gpumpsmap); i++ {
		gpumpsmap[i].RunFlag = 0
	}
	for i := 0; i < len(mps); i++ {
		for j := 0; j < len(gpumpsmap); j++ {
			if gpumpsmap[j].PID == mps[i].PID {
				gpumpsmap[j].RunFlag = 1
				break
			}
		}
	}
	for i := 0; i < len(gpumpsmap); i++ {
		if gpumpsmap[i].RunFlag == 0 {
			gpumpsmap[i].UUID = ""
			gpumpsmap[i].Container = ""
			gpumpsmap[i].useMemory = 0
			gpumpsmap[i].PID = 0
			gpumpsmap[i].Process = ""
			gpumpsmap[i].Index = 0
		}
	}*/
	//fmt.Println(gpumpsmap)

}
