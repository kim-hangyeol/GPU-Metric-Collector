package grpcs

type GrpcGPU struct {
	GrpcGPUUUID       string
	GrpcGPUused       int64
	GrpcGPUfree       int64
	GrpcGPUName       string
	GrpcGPUIndex      int
	GrpcGPUtotal      int64
	GrpcGPUtemp       int
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
}

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
	GrpcNodeUUID         []string
	NodeGPU              []*GrpcGPU
}

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
}
