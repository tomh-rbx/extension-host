// Copyright 2025 steadybit GmbH. All rights reserved.

package cpufreq

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCPUFrequencyInfo(t *testing.T) {
	tests := []struct {
		name       string
		contentMin string
		contentMax string
		wantMin    uint64
		wantMax    uint64
		wantErr    bool
	}{
		{
			name:       "valid min and max frequency",
			contentMin: "2200000",
			contentMax: "3400000",
			wantMin:    2200,
			wantMax:    3400,
			wantErr:    false,
		},
		{
			name:       "invalid min frequency",
			contentMin: "invalid",
			contentMax: "3400000",
			wantMin:    0,
			wantMax:    0,
			wantErr:    true,
		},
		{
			name:       "invalid max frequency",
			contentMin: "2200000",
			contentMax: "invalid",
			wantMin:    0,
			wantMax:    0,
			wantErr:    true,
		},
		{
			name:       "empty content",
			contentMin: "",
			contentMax: "",
			wantMin:    0,
			wantMax:    0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeCpuDirectory(t)
			fakeCpuFile(t, path.Join("cpu0", "cpufreq", "cpuinfo_min_freq"), tt.contentMin)
			fakeCpuFile(t, path.Join("cpu0", "cpufreq", "cpuinfo_max_freq"), tt.contentMax)

			gotMin, gotMax, err := GetCPUFrequencyInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCPUFrequencyInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotMin != tt.wantMin {
				t.Errorf("GetCPUFrequencyInfo() gotMin = %v, want %v", gotMin, tt.wantMin)
			}
			if gotMax != tt.wantMax {
				t.Errorf("GetCPUFrequencyInfo() gotMax = %v, want %v", gotMax, tt.wantMax)
			}
		})
	}
}

func fakeCpuDirectory(t *testing.T) {
	oldBasePath := cpuBasePath
	cpuBasePath = t.TempDir()

	t.Cleanup(func() {
		cpuBasePath = oldBasePath
	})
}

func fakeCpuFile(t *testing.T, name, content string) {
	if len(content) == 0 {
		return
	}

	err := os.MkdirAll(path.Join(cpuBasePath, path.Dir(name)), 0777)
	if err != nil {
		t.Error(err)
	}

	err = os.WriteFile(path.Join(cpuBasePath, name), []byte(content), 0666)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrentFrequency(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    uint64
		wantErr bool
	}{
		{
			name:    "valid current frequency",
			content: "2800000",
			want:    2800,
			wantErr: false,
		},
		{
			name:    "invalid current frequency",
			content: "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty content",
			content: "",
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeCpuDirectory(t)
			fakeCpuFile(t, path.Join("cpu0", "cpufreq", "scaling_cur_freq"), tt.content)

			got, err := GetCurrentFrequency()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentFrequency() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetCurrentFrequency() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetCPUFrequencyLimits(t *testing.T) {
	tests := []struct {
		name           string
		min            uint64
		max            uint64
		wantErr        string
		wantMinContent string
		wantMaxContent string
	}{
		{
			name:    "invalid min frequency (higher than max)",
			min:     1,
			max:     0,
			wantErr: "minimum frequency 1 MHz cannot be greater than maximum frequency 0 MHz",
		},
		{
			name:    "invalid min frequency (too low)",
			min:     2700,
			max:     3600,
			wantErr: "requested minimum frequency 2700 MHz is below hardware minimum 2800 MHz",
		},
		{
			name:    "invalid max frequency (too high)",
			min:     2800,
			max:     3700,
			wantErr: "requested maximum frequency 3700 MHz is above hardware maximum 3600 MHz",
		},
		{
			name:           "valid min and max frequency",
			min:            2900,
			max:            3000,
			wantErr:        "",
			wantMinContent: "2900000",
			wantMaxContent: "3000000",
		},
	}
	for _, tt := range tests {
		fakeCpuDirectory(t)
		for _, cpu := range []string{"cpu0", "cpu1"} {
			fakeCpuFile(t, path.Join(cpu, "cpufreq", "cpuinfo_min_freq"), "2800000")
			fakeCpuFile(t, path.Join(cpu, "cpufreq", "scaling_min_freq"), "2800000")
			fakeCpuFile(t, path.Join(cpu, "cpufreq", "cpuinfo_max_freq"), "3600000")
			fakeCpuFile(t, path.Join(cpu, "cpufreq", "scaling_max_freq"), "3600000")
		}

		t.Run(tt.name, func(t *testing.T) {
			if err := SetCPUFrequencyLimits(tt.min, tt.max); len(tt.wantErr) > 0 {
				if err.Error() != tt.wantErr {
					t.Errorf("SetCPUFrequencyLimits() error = %v, wantErr = %v", err, tt.wantErr)
				}
			} else {
				for _, cpu := range []string{"cpu0", "cpu1"} {
					gotMin, err := os.ReadFile(path.Join(cpuBasePath, cpu, "cpufreq", "scaling_min_freq"))
					require.NoError(t, err)
					assert.Equal(t, tt.wantMinContent, string(gotMin))

					gotMax, err := os.ReadFile(path.Join(cpuBasePath, cpu, "cpufreq", "scaling_max_freq"))
					require.NoError(t, err)
					assert.Equal(t, tt.wantMaxContent, string(gotMax))
				}
			}
		})
	}
}
