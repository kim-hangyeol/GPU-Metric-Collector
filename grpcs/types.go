package grpcs

type GrpcGPU struct {
	GrpcGPUUUID  string
	GrpcGPUused  uint64
	GrpcGPUfree  uint64
	GrpcGPUName  string
	GrpcGPUIndex int
	GrpcGPUtotal uint64
	GrpcGPUtemp  int
	GrpcGPUpower int
}

type GrpcNode struct {
	GrpcNodeName   string
	GrpcNodeCPU    int64
	GrpcNodeCount  int
	GrpcNodeMemory int64
	GrpcNodeUUID   []string
}
