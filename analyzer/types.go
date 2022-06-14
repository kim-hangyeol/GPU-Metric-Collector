package analyzer

type NodeMetric struct {
	AverCpu       int64
	AverMemory    int64
	AverStorage   int64
	AverNetworkRx int64
	AverNetworkTx int64
}

type GPUMetric struct {
	AverMemory   int64
	AverUtil     int
	AverFanSpeed int
	AverPower    int
	AverTemp     int
	AverRX       int64
	AverTX       int64
}

type PodMetric struct {
	AverMemory    int64
	AverGPUMemory int64
	AverCPU       float64
	AverStorage   int64
	AverNetworkRx int64
	AverNetworkTx int64
}

type Degradation struct {
	NodeName      string
	ISIncrease    bool
	ISDegradation bool
	GPU           []*DegradationGPU
}

type DegradationGPU struct {
	GPUUUID       string
	ISDegradation bool
	ISIncrease    bool
	Pod           []*DegradationPod
}

type DegradationPod struct {
	PodUID        string
	ISIncrease    bool
	ISDegradation bool
}
