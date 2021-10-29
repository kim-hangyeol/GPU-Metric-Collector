package grpcs

type GrpcGPU struct {
	GrpcGPUUUID   string
	GrpcGPUMemory uint64
	GrpcGPUName   string
	GrpcGPUIndex  int
	GrpcGPUfull   uint64
	GrpcGPUtemp   int
	GrpcGPUpower  int
}

type GrpcNode struct {
	GrpcNodeName   string
	GrpcNodeCPU    int64
	GrpcNodeCount  int
	GrpcNodeMemory int64
	GrpcNodeUUID   []string
}
