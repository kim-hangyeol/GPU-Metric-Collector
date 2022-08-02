package analyzer

type NodeMetric struct {
	AverCpu        int64
	AverMemory     int64
	AverStorage    int64
	AverNetworkRx  int64
	AverNetworkTx  int64
	StDevCpu       float64
	StDevMemory    float64
	StDevStorage   float64
	StDevNetworkRx float64
	StDevNetworkTx float64
}

type GPUMetric struct {
	AverMemory    int64
	AverUtil      int
	AverFanSpeed  int
	AverPower     int
	AverTemp      int
	AverRX        int64
	AverTX        int64
	StDevMemory   float64
	StDevUtil     float64
	StDevFanSpeed float64
	StDevPower    float64
	StDevTemp     float64
	StDevRX       float64
	StDevTX       float64
}

type PodMetric struct {
	AverMemory     int64
	AverGPUMemory  int64
	AverCPU        float64
	AverStorage    int64
	AverNetworkRx  int64
	AverNetworkTx  int64
	StDevMemory    float64
	StDevGPUMemory float64
	StDevCPU       float64
	StDevStorage   float64
	StDevNetworkRx float64
	StDevNetworkTx float64
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

type Regression struct {
	Memory      []float64
	GPUMemory   []float64
	CPU         []float64
	Storage     []float64
	RX          []float64
	TX          []float64
	Power       []float64
	Temperature []float64
	Fanspeed    []float64
	Utilization []float64
}
