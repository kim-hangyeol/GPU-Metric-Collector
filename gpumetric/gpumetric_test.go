package gpumetric

import (
	"testing"
)

func TestGpumetric(t *testing.T) {

	first := struct {
		firstgpu  *nvml1
		secondgpu *nvml1
	}{
		firstgpu: &nvml1{
			Init:           func() error { return nil },
			Shutdown:       func() error { return nil },
			GetDeviceCount: func() (uint, error) { return 1, nil },
			NewDevice: func(uint) (Device, error) {
				return Device{
					Clocks:      2100,
					UUID:        "123-123-123-123-123",
					Path:        "123-123",
					Model:       "NVIDIA GeForce RTX 3080",
					Power:       340,
					Memory:      10018,
					CPUAffinity: 123,
					Status: func() (Stat, error) {
						return Stat{
							Memory:      0,
							Power:       20,
							Temperature: 30,
							Clocks:      0,
							PCI: PCIStatusInfo{
								BAR1Used: 0,
								Throughput: PCIThroughputInfo{
									RX: 0,
									TX: 0,
								},
							},
						}, nil
					},
				}, nil
			},
		},
		secondgpu: &nvml1{
			Init:           func() error { return nil },
			Shutdown:       func() error { return nil },
			GetDeviceCount: func() (uint, error) { return 1, nil },
			NewDevice: func(uint) (Device, error) {
				return Device{
					Clocks:      2100,
					UUID:        "123-123-123-123-123",
					Path:        "123-123",
					Model:       "NVIDIA GeForce RTX 3080",
					Power:       340,
					Memory:      10018,
					CPUAffinity: 123,
					Status: func() (Stat, error) {
						return Stat{
							Memory:      5000,
							Power:       50,
							Temperature: 60,
							Clocks:      1500,
							PCI: PCIStatusInfo{
								BAR1Used: 100,
								Throughput: PCIThroughputInfo{
									RX: 1000,
									TX: 2000,
								},
							},
						}, nil
					},
				}, nil
			},
		},
	}
	//Gpumetric(*first.firstgpu)
	Gpumetric(*first.secondgpu)

}

