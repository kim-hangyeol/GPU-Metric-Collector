module metric-collector

go 1.16

require (
	github.com/Masterminds/semver v1.5.0
	github.com/NVIDIA/go-dcgm v0.0.0-20210825154740-df776fdbfea0 // indirect
	github.com/NVIDIA/go-nvml v0.11.6-0
	github.com/NVIDIA/gpu-monitoring-tools v0.0.0-20210803220325-7c362b2f4913 // indirect
	github.com/influxdata/influxdb1-client v0.0.0-20200827194710-b269163b24ab
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
)
