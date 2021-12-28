package grpcs

type GrpcGPU struct {
	GrpcGPUUUID       string
	GrpcGPUused       uint64
	GrpcGPUfree       uint64
	GrpcGPUName       string
	GrpcGPUIndex      int
	GrpcGPUtotal      uint64
	GrpcGPUtemp       int
	GrpcGPUpower      int
	GrpcGPUmpscount   int
	GrpcGPUtotalpower int
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
	GrpcNodeUUID         []string
}
