package analyzer

// 노드 분석을 위해 노드메트릭 가져오는 구조체
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
	MemoryList     []float64
	CPUList        []float64
	StorageList    []float64
	RXList         []float64
	TXList         []float64
}

// GPU 분석을 위해 GPU 메트릭 가져오는 구조체
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
	GPUUtilList   []float64
	FanSpeedList  []float64
	GPUMemoryList []float64
	PowerList     []float64
	RXList        []float64
	TXList        []float64
}

// Pod 분석을 위해 Pod 메트릭 가져오는 구조체
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
	MemoryList     []float64
	GPUMemoryList  []float64
	CPUList        []float64
	StorageList    []float64
	RXList         []float64
	TXList         []float64
}

// 스케줄러로 성능저하 알릴때 보내는 구조체
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
