package gpumpsmetric

import (
	"context"
	"fmt"
	"log"
	"metric-collector/storage"
	"os"
	"strconv"
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

func Gpumpsmetric(device nvml.Device, count int, c influxdb.Client, data *storage.Collection) {
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
	fmt.Println(runpodlist)
	fmt.Println(data.Metricsbatchs[0].Pods)

	for i := 1; i < len(mps); i++ {
		if mps[i].Name != "nvidia-cuda-mps-server" {
			//fmt.Printf("MPS |%4v| : |%10vMiB| |%30v|  |%20v|\n", i-1, mps[i].MemoryUsed, gpumpsmap[i-1].Pod, gpumpsmap[i-1].Container)
			bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
				Database:  "metric",
				Precision: "s",
			})

			tags := map[string]string{"UUID": UUID}
			fields := map[string]interface{}{
				"gpu_mps_count":   strconv.Itoa(len(mps) - 1),
				"gpu_mps_index":   strconv.Itoa(i),
				"gpu_mps_process": mps[i].Name,
				"gpu_mps_memory":  strconv.Itoa(int(mps[i].MemoryUsed)),
			}
			pt, err := influxdb.NewPoint("gpumap", tags, fields, time.Now())
			if err != nil {
				fmt.Println("Error:", err.Error())
			}
			bp.AddPoint(pt)
			err = c.Write(bp)
			if err != nil {
				fmt.Println("Error:", err.Error())
			}
		} else {
			fmt.Printf("MPS |%4v| : |%10vMiB| |%30v|  |%20v|\n", i-1, mps[i-1].MemoryUsed, gpumpsmap[i-1].Pod, gpumpsmap[i-1].Container)
			bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
				Database:  "metric",
				Precision: "s",
			})

			tags := map[string]string{"UUID": UUID}
			fields := map[string]interface{}{
				"gpu_mps_count":   strconv.Itoa(len(mps) - 1),
				"gpu_mps_index":   strconv.Itoa(i),
				"gpu_mps_process": mps[i-1].Name,
				"gpu_mps_memory":  strconv.Itoa(int(mps[i-1].MemoryUsed)),
			}
			pt, err := influxdb.NewPoint("gpumap", tags, fields, time.Now())
			if err != nil {
				fmt.Println("Error:", err.Error())
			}
			bp.AddPoint(pt)
			err = c.Write(bp)
			if err != nil {
				fmt.Println("Error:", err.Error())
			}
		}
		// if mps[i].Name != "nvidia-cuda-mps-server" {
		// 	//fmt.Printf("MPS |%4v| : |%10vMiB| |%30v|  |%20v|\n", i-1, mps[i].MemoryUsed, gpumpsmap[i-1].Pod, gpumpsmap[i-1].Container)
		// 	bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		// 		Database:  "metric",
		// 		Precision: "s",
		// 	})

		// 	tags := map[string]string{"UUID": UUID}
		// 	fields := map[string]interface{}{
		// 		"nodename":        runpodlist[i].Spec.NodeName,
		// 		"podname":   runpodlist[i].Name,
		// 		"maxgpumem": ,
		// 		"avergpumem":  ,
		// 		"avercpu": ,
		// 		"aversysmem": ,
		// 		"userinput": ,
		// 	}
		// 	pt, err := influxdb.NewPoint("podrecord", tags, fields, time.Now())
		// 	if err != nil {
		// 		fmt.Println("Error:", err.Error())
		// 	}
		// 	bp.AddPoint(pt)
		// 	err = c.Write(bp)
		// 	if err != nil {
		// 		fmt.Println("Error:", err.Error())
		// 	}
		// } else {
		// 	fmt.Printf("MPS |%4v| : |%10vMiB| |%30v|  |%20v|\n", i-1, mps[i-1].MemoryUsed, gpumpsmap[i-1].Pod, gpumpsmap[i-1].Container)
		// 	bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		// 		Database:  "metric",
		// 		Precision: "s",
		// 	})

		// 	tags := map[string]string{"UUID": UUID}
		// 	fields := map[string]interface{}{
		// 		"nodename":        runpodlist[i].Spec.NodeName,
		// 		"podname":   runpodlist[i].Name,
		// 		"maxgpumem": ,
		// 		"avergpumem":  ,
		// 		"avercpu": ,
		// 		"aversysmem": ,
		// 		"userinput": ,
		// 	}
		// 	pt, err := influxdb.NewPoint("podrecord", tags, fields, time.Now())
		// 	if err != nil {
		// 		fmt.Println("Error:", err.Error())
		// 	}
		// 	bp.AddPoint(pt)
		// 	err = c.Write(bp)
		// 	if err != nil {
		// 		fmt.Println("Error:", err.Error())
		// 	}
		// }
	}

}
