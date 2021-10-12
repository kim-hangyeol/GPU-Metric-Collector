// Copyright 2018 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"k8s.io/apimachinery/pkg/api/resource"
)

type Collection struct {
	Metricsbatchs []MetricsBatch
	ClusterName   string
	LatencyTime   *timestamppb.Timestamp
}

// MetricsBatch is a single batch of pod, container, and node metrics from some source.
type MetricsBatch struct {
	IP   string
	Node NodeMetricsPoint
	Pods []PodMetricsPoint
}

// NodeMetricsPoint contains the metrics for some node at some point in time.
type NodeMetricsPoint struct {
	Name string
	MetricsPoint
}

// PodMetricsPoint contains the metrics for some pod's containers.
type PodMetricsPoint struct {
	Name      string
	Namespace string
	MetricsPoint
	Containers []ContainerMetricsPoint
}

// ContainerMetricsPoint contains the metrics for some container at some point in time.
type ContainerMetricsPoint struct {
	Name string
	MetricsPoint
}

// MetricsPoint represents the a set of specific metrics at some point in time.
type MetricsPoint struct {
	Timestamp time.Time

	// Cpu
	CPUUsageNanoCores resource.Quantity

	// Memory
	MemoryUsageBytes      resource.Quantity
	MemoryAvailableBytes  resource.Quantity
	MemoryWorkingSetBytes resource.Quantity

	// Network
	NetworkRxBytes resource.Quantity
	NetworkTxBytes resource.Quantity

	// Fs
	FsAvailableBytes resource.Quantity
	FsCapacityBytes  resource.Quantity
	FsUsedBytes      resource.Quantity
}

type MultiMetric struct {
	cpu_usage int
	cpu_total int
	cpu_temp int
	memory_usage int
	memory_total int
	network_rx_usage int
	network_tx_usage int
	gpu_count int
	node_name string
}

type PodRecord struct {
	gpu_uuid string
	pod_name string
	node_name string
	pod_max_gpumemory int
	pod_average_gpumemory float64
	pod_average_cpu float64
	pod_max_memory int
	user_input UserInput
}


type GPUMetric struct {
	pcie_rx_usage int
	pcie_tx_usage int
	gpu_mps_count int
	gpu_mps_max int
	gpu_name string
	gpu_index int
	gpu_utilization int
	gpu_uuid string
	gpu_memory int
	gpu_power int
	gpu_temp int
}

type GPUMap struct {
	gpu_uuid string
	gpu_mps_index int
	gpu_mps_process string
	gpu_mps_memory int
}


type UserInput struct {

}

type CPUResource struct {

}

type GPUResource struct {

}