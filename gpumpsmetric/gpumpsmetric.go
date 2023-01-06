package gpumpsmetric

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"metric-collector/grpcs"
	"metric-collector/storage"
	"os"
	"strings"
	"time"

	//"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
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

// gpu파드+쿠베파드 매핑 (temp)
type GpuMpsMap struct {
	UUID string
	// useMemory int
	UID         string
	Container   string
	ContainerID string
	Pod         string
	PID         uint
	Index       int
	RunFlag     int
	StartTime   time.Time
}

type processinfo struct {
	nvmlprocess []nvml.ProcessInfo
	Name        []string
	ContainerID []string
}

func getpodlist() (*v1.PodList, error) {
	//노드의 파드 리스트 가져오는것
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

func getgpuprocess(device nvml.Device, mps *processinfo) int {
	//gpu에서 동작중인 프로세스 가져오는 것
	mpscount := 0
	mps.nvmlprocess, _ = device.GetMPSComputeRunningProcesses()
	mps.Name = nil
	mps.ContainerID = nil
	for i := 0; i < len(mps.nvmlprocess); i++ {
		// GPU_Metric.GPUPod = append(GPU_Metric.GPUPod, &grpcs.PodMetric{})
		// fmt.Println(mps.nvmlprocess[i].Pid)
		pid := fmt.Sprint(mps.nvmlprocess[i].Pid)
		cgroupfile, _ := os.Open("/proc/" + pid + "/cgroup")
		cgroupscanner := bufio.NewScanner(cgroupfile)
		for cgroupscanner.Scan() {
			cgroup := cgroupscanner.Text()
			cgroups := strings.Split(cgroup, "/")
			// fmt.Println(cgroups)
			if cgroups[1] == "kubepods" || cgroups[1] == "kubepods.slice" {
				if len(cgroups) > 3 {
					tmpslice := strings.Split(cgroups[4], "-")
					if len(tmpslice) > 1 {
						tmpslice1 := strings.Split(tmpslice[1], ".")
						mps.ContainerID = append(mps.ContainerID, tmpslice1[0])
						mpscount++

					} else {
						mps.ContainerID = append(mps.ContainerID, cgroups[4])
						mpscount++
					}
					break
				} else {
					mps.ContainerID = append(mps.ContainerID, "")
				}
			} else {
				continue
			}
		}
		processname, _ := nvml.SystemGetProcessName(int(mps.nvmlprocess[i].Pid))
		mps.Name = append(mps.Name, processname)
		// GPU_Metric.GPUPod[i].ProcessName = processname
	}
	// fmt.Println(mpscount)
	return mpscount
}

// var firstnum = 0

func Gpumpsmetric(device nvml.Device, count int, c influxdb.Client, data *storage.Collection, podmap *[]storage.PodGPU, GPU_Metric *grpcs.GrpcGPU) int {
	//메인 동작
	var gpumpsmap [48]GpuMpsMap //최대 48개
	var mps processinfo         //gpu 프로세스 정보

	//var pidtable []uint
	//var mapping [10]int
	UUID, _ := device.GetUUID()
	// fmt.Println(UUID)
	// mpscount := getgpuprocess(device, &mps)

	//fmt.Println(mps)
	// if err != nvml.SUCCESS {
	// 	log.Fatalln(err)
	// }
	var runpodlist []v1.Pod
	var runpod string

	for loopnum := 0; loopnum < 20; loopnum++ {
		if loopnum != 0 {
			time.Sleep(time.Millisecond)
		}
		podnum := 0
		mpscount := getgpuprocess(device, &mps)
		// fmt.Println(mps)
		podlist, err := getpodlist() //해당노드/네임스페이스
		if err != nil {
			log.Fatalln(err)
		}
		// fmt.Println(podlist)
		for i := 0; i < len(podlist.Items); i++ { //container creating 때문에 갯수가 안맞는듯?->그래서 20번 반복
			annotation := podlist.Items[i].Annotations["UUID"]
			annotationUUID := strings.Split(annotation, ",")
			if len(annotationUUID) > 1 {
				for j := 0; j < len(annotationUUID); j++ {
					if annotationUUID[j] == UUID {
						// gpumpsmap[podnum].Container = podlist.Items[i].Spec.Containers[0].Name
						// gpumpsmap[podnum].Pod = podlist.Items[i].ObjectMeta.Name
						podnum++
					}
				}
			} else {
				if podlist.Items[i].Annotations["UUID"] == UUID {
					// fmt.Println(podlist.Items[i])
					//fmt.Printf("running pod name : %v\n", podlist.Items[i].ObjectMeta.Name)
					// gpumpsmap[podnum].Container = podlist.Items[i].Spec.Containers[0].Name
					// gpumpsmap[podnum].Pod = podlist.Items[i].ObjectMeta.Name
					podnum++
					// fmt.Println(podnum)
				}
				// fmt.Println(podnum)
			}
			// fmt.Println(podnum)
		}
		// fmt.Println(podnum)
		// fmt.Println(podnum)
		// fmt.Println(mpscount)
		if podnum == mpscount && podnum != 0 {
			runpodlist = append(runpodlist, podlist.Items...)
			//fmt.Printf("running pod num : %v\nrunning process num : %v\n", podnum, len(mps))
			//runpodlist = *podlist
			break
		} else if podnum == 0 && mpscount == 0 {
			break
		}
	}
	// for i := 0; i < len(runpodlist); i++ {
	// 	for j := 0; j < len(runpodlist); j++ {
	// 		if runpodlist[i].Status.StartTime.Before(runpodlist[j].Status.StartTime) {
	// 			temppod = runpodlist[i]
	// 			runpodlist[i] = runpodlist[j]
	// 			runpodlist[j] = temppod
	// 		}
	// 	}
	// }
	podnum := 0
	// fmt.Println(runpodlist)
	for i := 0; i < len(runpodlist); i++ {
		annotation := runpodlist[i].Annotations["UUID"]
		// fmt.Println(annotation)
		annotationUUID := strings.Split(annotation, ",")
		// fmt.Println(annotationUUID)
		// fmt.Println(runpodlist[i].Annotations["UUID"], " ", UUID)
		if len(annotationUUID) > 1 { //UUID 안봐도 될듯??
			for j := 0; j < len(annotationUUID); j++ {
				if annotationUUID[j] == UUID {
					gpumpsmap[podnum].UID = string(runpodlist[i].UID)
					gpumpsmap[podnum].Container = runpodlist[i].Spec.Containers[0].Name
					gpumpsmap[podnum].Pod = runpodlist[i].ObjectMeta.Name
					gpumpsmap[podnum].StartTime = runpodlist[i].Status.ContainerStatuses[0].State.Running.StartedAt.Time.Local()
					gpumpsmap[podnum].ContainerID = runpodlist[i].Status.ContainerStatuses[0].ContainerID
					runpod = runpod + runpodlist[i].ObjectMeta.Name
					podnum++
				}
			}
		} else {
			if runpodlist[i].Annotations["UUID"] == UUID {
				// fmt.Printf("running pod name : %v\n", runpodlist[i].Name)
				gpumpsmap[podnum].UID = string(runpodlist[i].UID)
				gpumpsmap[podnum].Container = runpodlist[i].Spec.Containers[0].Name
				gpumpsmap[podnum].Pod = runpodlist[i].ObjectMeta.Name
				gpumpsmap[podnum].StartTime = runpodlist[i].Status.ContainerStatuses[0].State.Running.StartedAt.Time.Local()
				gpumpsmap[podnum].ContainerID = runpodlist[i].Status.ContainerStatuses[0].ContainerID
				runpod = runpod + runpodlist[i].ObjectMeta.Name
				podnum++
			}
		}
	}

	//슬럼쪽 코드
	// fmt.Println(podnum)
	// for i := 0; i < podnum; i++ {
	// 	for j := i + 1; j < podnum; j++ {
	// 		if gpumpsmap[i].StartTime.Before(gpumpsmap[j].StartTime) {
	// 			temppod := gpumpsmap[i]
	// 			gpumpsmap[i] = gpumpsmap[j]
	// 			gpumpsmap[j] = temppod
	// 		}
	// 		//fmt.Println(runpodlist[j].Status.ContainerStatuses[0].State.Running.StartedAt.Time.Local().Unix())
	// 	}
	// }
	// fmt.Println(podnum)
	//fmt.Println("podnum = ", podnum)
	// var timetable []int // 1은 쿠버 2는 슬럼
	// kube := 0
	// slurm := 0
	// for i := 0; i < podnum+len(slurmjob); i++ {
	// 	if kube == podnum {
	// 		timetable = append(timetable, 2)
	// 	} else if slurm == len(slurmjob) {
	// 		timetable = append(timetable, 1)
	// 	} else {
	// 		slurmtime, _ := strconv.Atoi(slurmjob[slurm].StartTime)
	// 		if gpumpsmap[kube].StartTime > int64(slurmtime) {
	// 			fmt.Println(gpumpsmap[kube].StartTime, "asd", slurmjob[slurm].StartTime)
	// 			timetable = append(timetable, 2)
	// 			slurm++
	// 		} else {
	// 			timetable = append(timetable, 1)
	// 			kube++
	// 		}
	// 	}
	// }
	// fmt.Println(timetable)
	//fmt.Println(runpodlist)
	//fmt.Println(mps)

	// for i := 0; i < len(podmap); i++ {
	// 	podmap[i].Isrunning = 0
	// }

	//fmt.Println(gpumpsmap)
	//var isnewpod = 1
	//var newpodcount = 0

	//GPU 프로세스랑 Container랑 매핑하는 부분
	for i := 0; i < podnum; i++ {
		for k := 0; k < len(mps.ContainerID); k++ {
			if mps.ContainerID[k] == "" {
				continue
			}
			//컨테이너 아이디 확인한걸로 로그 파일 접근 가능
			//로깅하는 코드는 5.66:/root/workspace/kmc/logmanager에 존재
			//코드 따로 짠 이유는 메콜은 수집주기가 너무 길어서임
			containerid := strings.Split(gpumpsmap[i].ContainerID, "/")[2]
			if mps.ContainerID[k] == containerid {
				GPU_Metric.GPUPod = append(GPU_Metric.GPUPod, &grpcs.PodMetric{})
				GPU_Metric.GPUPod[i].PodGPUMemory = int64(mps.nvmlprocess[k].UsedGpuMemory)
				GPU_Metric.GPUPod[i].PodPid = mps.nvmlprocess[k].Pid
				GPU_Metric.GPUPod[i].PodName = gpumpsmap[i].Pod
				GPU_Metric.GPUPod[i].ProcessName = mps.Name[k]
				GPU_Metric.GPUPod[i].PodUid = gpumpsmap[i].UID
				GPU_Metric.GPUPod[i].ContainerID = containerid
				for j := 0; j < len(data.Metricsbatchs[0].Pods); j++ {
					if GPU_Metric.GPUPod[i].PodName == data.Metricsbatchs[0].Pods[j].Name {
						GPU_Metric.GPUPod[i].PodCPU = float64(data.Metricsbatchs[0].Pods[j].CPUUsageNanoCores.MilliValue())
						GPU_Metric.GPUPod[i].PodStorage = int64(data.Metricsbatchs[0].Pods[j].FsUsedBytes.Value())
						GPU_Metric.GPUPod[i].PodNetworkTX = int64(data.Metricsbatchs[0].Pods[j].NetworkTxBytes.Value())
						GPU_Metric.GPUPod[i].PodNetworkRX = int64(data.Metricsbatchs[0].Pods[j].NetworkRxBytes.Value())
						GPU_Metric.GPUPod[i].PodMemory = int64(data.Metricsbatchs[0].Pods[j].MemoryUsageBytes.Value())
					}
				}
				break
			}
		}

		// GPU_Metric.GPUPod[i].PodMemory =
	}

	for i := 0; i < len(*podmap); i++ {
		(*podmap)[i].RunningCheck = false
	}

	for i := 0; i < len(GPU_Metric.GPUPod); i++ {
		newflag := true
		for j := 0; j < len(*podmap); j++ {
			if GPU_Metric.GPUPod[i].ContainerID == (*podmap)[j].ContainerID {
				(*podmap)[j].RunningCheck = true //아직동작
				newflag = false                  //원래있던거
				break
			}
		}
		if newflag {
			GPU_Metric.GPUAssingment = GPU_Metric.GPUAssingment + 1
			var tmppod storage.PodGPU
			tmppod.ContainerID = GPU_Metric.GPUPod[i].ContainerID
			(*podmap) = append((*podmap), tmppod)
		}
	}

	for i := 0; i < len(*podmap); i++ {
		if !(*podmap)[i].RunningCheck {
			if (*podmap)[i].HealthCheck {
				//정상 종료인지 확인을 해야함 --> Log Collector랑 연계 해야할듯
				GPU_Metric.GPUReturn = GPU_Metric.GPUReturn + 1
			}
			//podmap 삭제
		}
	}

	//매핑하는거 이전 버전
	// for i := 0; i < podnum; i++ {
	// 	if len(podmap) < podnum {
	// 		podmap = append(podmap, &storage.PodGPU{})
	// 	}
	// 	podmap[i].PodName = gpumpsmap[i].Pod
	// 	podmap[i].Isrunning = 1
	// 	podmap[i].Index = i
	// 	podmap[i].Pid = uint(mps.nvmlprocess[i].Pid)
	// 	podmap[i].Usegpumemory = int(mps.nvmlprocess[i].UsedGpuMemory)
	// }
	// for i := 0; i < podnum; i++ {
	// 	for j := 0; j < len(data.Metricsbatchs[0].Pods); j++ {
	// 		if podmap[i].PodName == data.Metricsbatchs[0].Pods[j].Name {
	// 			podmap[i].CPU = int(data.Metricsbatchs[0].Pods[j].CPUUsageNanoCores.Value())
	// 			podmap[i].Storage = int(data.Metricsbatchs[0].Pods[j].FsUsedBytes.Value())
	// 			podmap[i].NetworkTX = int(data.Metricsbatchs[0].Pods[j].NetworkTxBytes.Value())
	// 			podmap[i].NetworkRX = int(data.Metricsbatchs[0].Pods[j].NetworkRxBytes.Value())
	// 			podmap[i].Memory = int(data.Metricsbatchs[0].Pods[j].MemoryUsageBytes.Value())
	// 		}
	// 	}
	// }

	// for i := 0; i < podnum; i++ {
	// 	isnewpod = 1
	// 	// if len(podmap) == 0 {
	// 	// 	podmap = append(podmap, storage.PodGPU{})
	// 	// 	podmap[i].PodName = gpumpsmap[0].Pod
	// 	// 	//fmt.Println(mps[0].Name)
	// 	// 	podmap[i].Index = 0
	// 	// 	podmap[i].Isrunning = 1
	// 	// } else {
	// 	if i != podnum-1 {
	// 		for j := 0; j < len(podmap); j++ {
	// 			if gpumpsmap[i].Pod == podmap[j].PodName {
	// 				podmap[j].Isrunning = 1
	// 				continue
	// 			}
	// 		}
	// 	} else {
	// 		for j := 0; j < len(podmap); j++ {
	// 			if gpumpsmap[i].Pod == podmap[j].PodName {
	// 				isnewpod = 0
	// 				podmap[j].Isrunning = 1
	// 			}
	// 		}
	// 		if isnewpod == 1 {
	// 			podmap = append(podmap, storage.PodGPU{})
	// 			podmap[len(podmap)-1].PodName = gpumpsmap[0+newpodcount].Pod
	// 			podmap[len(podmap)-1].Isrunning = 1
	// 			podmap[len(podmap)-1].Index = 0 + newpodcount
	// 			podmap[len(podmap)-1].Pid = uint(mps.nvmlprocess[0+newpodcount].Pid)
	// 			newpodcount++
	// 		}
	// 	}
	// 	// }
	// }
	// var deletecount = 0
	// if podnum == 0 {
	// 	podmap = nil
	// } else {
	// 	for i := 0; i < len(podmap); i++ {
	// 		if podmap[i].Isrunning == 0 && i != 0 && i != len(podmap)-1 {
	// 			podmap = append(podmap[:i], podmap[i+1:]...)
	// 			deletecount++
	// 		}
	// 		if isnewpod == 1 && i == len(podmap)-1 {
	// 			continue
	// 		} else {
	// 			podmap[i].Index = podmap[i].Index - deletecount
	// 			podmap[i].Usegpumemory = int(mps[podmap[i].Index].MemoryUsed)
	// 		}
	// 	}
	// }
	// for i := 0; i < len(podmap); i++ {
	// 	for j := 0; j < len(mps.nvmlprocess); j++ {
	// 		if podmap[i].Pid != uint(mps.nvmlprocess[j].Pid) {
	// 			continue
	// 		} else {
	// 			podmap[i].Index = j
	// 			podmap[i].Usegpumemory = int(mps.nvmlprocess[j].UsedGpuMemory)
	// 			break
	// 		}
	// 	}
	// }
	//index, _ := device.GetIndex()
	//fmt.Println("gpu index : ", index)
	//fmt.Println(podmap)
	//fmt.Println(podmap)
	//fmt.Println(runpodlist)
	//fmt.Println(data.Metricsbatchs[0].Pods)

	for i := 0; i < podnum; i++ {
		//fmt.Printf("MPS |%4v| : |%10vMiB| |%30v|  |%20v|\n", i-1, mps[i-1].MemoryUsed, gpumpsmap[i-1].Pod, gpumpsmap[i-1].Container)
		// fmt.Println(GPU_Metric.GPUPod[i].ProcessName)
		bp, _ := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database:  "multimetric",
			Precision: "s",
		})

		tags := map[string]string{"UUID": UUID}
		fields := map[string]interface{}{
			// "gpu_mps_count":   strconv.Itoa(len(mps.nvmlprocess) - 1),
			// "gpu_mps_index":   strconv.Itoa(i),
			"gpu_mps_process":    GPU_Metric.GPUPod[i].ProcessName,
			"gpu_mps_memory":     GPU_Metric.GPUPod[i].PodGPUMemory,
			"gpu_mps_pid":        GPU_Metric.GPUPod[i].PodPid,
			"gpu_mps_pod":        GPU_Metric.GPUPod[i].PodName,
			"gpu_mps_networkrx":  GPU_Metric.GPUPod[i].PodNetworkRX,
			"gpu_mps_networktx":  GPU_Metric.GPUPod[i].PodNetworkTX,
			"gpu_mps_storage":    GPU_Metric.GPUPod[i].PodStorage,
			"gpu_mps_cpu":        GPU_Metric.GPUPod[i].PodCPU,
			"gpu_mps_nodememory": GPU_Metric.GPUPod[i].PodMemory,
			"gpu_mps_uid":        GPU_Metric.GPUPod[i].PodUid,
		}
		pt, err := influxdb.NewPoint("podmetric", tags, fields, time.Now())
		if err != nil {
			fmt.Println("Error:", err.Error())
		}
		bp.AddPoint(pt)
		err = c.Write(bp)
		if err != nil {
			fmt.Println("Error:", err.Error())
		}

		//프린트 찍은거
		// fmt.Println("--------------GPU Map--------------")
		// fmt.Println("GPU UUID : ", UUID)
		// fmt.Println("Process Name : ", mps[i].Name)
		// fmt.Println("Process Pid : ", mps[i].PID)
		// fmt.Println("Process Memory (Used) : ", mps[i].MemoryUsed)
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
	// a, _ := device.GetAccountingPids()
	// for i := 0; i < len(a); i++ {
	// 	fmt.Println(device.GetAccountingStats(uint32(a[i])))
	// }
	return podnum
}
