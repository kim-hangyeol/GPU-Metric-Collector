package grpcs

//총집합(보내기위함)

// GPU 메트릭 구조체
type GrpcGPU struct {
	GrpcGPUUUID       string
	GrpcGPUtotal      int64
	GrpcGPUfree       int64
	GrpcGPUused       int64
	GrpcGPUName       string
	GrpcGPUIndex      int
	GrpcGPUtemp       GPUTemperature
	GrpcGPUpower      int
	GrpcGPUmpscount   int
	GrpcGPUtotalpower int
	GrpcGPUflops      int
	GrpcGPUarch       int
	GrpcGPUutil       int
	FanSpeed          int
	GPUPod            []*PodMetric
	GPURX             int
	GPUTX             int
	GPUAssingment     int
	GPUReturn         int
}

// Node 메트릭 구조체
type GrpcNode struct {
	GrpcNodeName         string
	GrpcNodetotalCPU     int64
	GrpcNodeCPU          int64
	GrpcNodeCount        int
	GrpcNodeTotalMemory  int64
	GrpcNodeMemory       int64
	GrpcNodeTotalStorage int64
	GrpcNodeStorage      int64
	NodeNetworkRX        int64
	NodeNetworkTX        int64
	TotalPodnum          int
	GrpcNodeUUID         []string
	NodeGPU              []*GrpcGPU
	NvLinkInfo           []NvLink
}

// Pod 메트릭 구조체
type PodMetric struct {
	PodUid       string
	PodName      string
	ProcessName  string
	PodPid       uint32
	PodGPUMemory int64
	PodMemory    int64
	PodCPU       float64
	PodNetworkRX int64
	PodNetworkTX int64
	PodStorage   int64
	ContainerID  string
}

// Node 메트릭 내부 nvlink 구조체
type NvLink struct {
	GPU1UUID  string
	GPU2UUID  string
	CountLink int
}

// GPU 메트릭 내부 gputemp 구조체
type GPUTemperature struct {
	Current      int //현재 온도
	Threshold    int //성능 저하 온도
	Shutdown     int //강제 종료 온도
	MaxOperating int //?
}
