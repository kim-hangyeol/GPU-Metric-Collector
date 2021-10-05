package gpumpsmetric

import (
	"testing"
)

func TestGpumetric(t *testing.T) {

	first := struct {
		firstgpu  *Device
		secondgpu *Device
	}{
		firstgpu: &Device{
			UUID: "GPU-123-123-123-123",
			GetAllRunningProcesses: func() ([]ProcessInfo, error) {
				return []ProcessInfo{}, nil
			},
		},
		secondgpu: &Device{
			UUID: "GPU-123-123-123-123",
			GetAllRunningProcesses: func() ([]ProcessInfo, error) {
				return []ProcessInfo{
					{
						PID:        111111,
						Name:       "test",
						MemoryUsed: 2000,
						Type:       1,
					},
					{
						PID:        222222,
						Name:       "TEST",
						MemoryUsed: 3000,
						Type:       0,
					},
				}, nil
			},
		},
	}
	Gpumpsmetric(*first.firstgpu)
	//Gpumpsmetric(*first.secondgpu)

}

