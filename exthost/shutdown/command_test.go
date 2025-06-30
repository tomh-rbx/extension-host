// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package shutdown

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func Test_commandShutdown_IsAvailable(t *testing.T) {
	tests := []struct {
		name        string
		lookPathErr error
		want        bool
	}{
		{
			name: "should return true when shutdown is available and executable",
			want: true,
		},
		{
			name:        "should return false when shutdown is not found",
			lookPathErr: errors.New("test error"),
			want:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &commandShutdown{lookPath: func(name string) (string, error) {
				return "", tt.lookPathErr
			}}
			if got := c.IsAvailable(); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_commandShutdown_Reboot(t *testing.T) {
	tests := []struct {
		name          string
		run           func(name string, arg ...string) error
		wantErr       bool
		expectedCalls [][]string
	}{
		{
			name: "should report error when dry-run fails",
			run: func(name string, arg ...string) error {
				return errors.New("test error")
			},
			wantErr: true,
			expectedCalls: [][]string{
				{"shutdown", "-k", "--no-wall", "now"},
				{"shutdown", "-c"},
			},
		},
		{
			name: "should not report error when dry-run succeeds",
			run: func(name string, arg ...string) error {
				return nil
			},
			wantErr: false,
			expectedCalls: [][]string{
				{"shutdown", "-k", "--no-wall", "now"},
				{"shutdown", "-r", "now"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := make([][]string, 0)
			var m sync.Mutex

			c := &commandShutdown{
				run: func(ctx context.Context, name string, arg ...string) error {
					m.Lock()
					defer m.Unlock()
					calls = append(calls, append([]string{name}, arg...))
					return tt.run(name, arg...)
				},
			}

			if err := c.Reboot(); (err != nil) != tt.wantErr {
				t.Errorf("Reboot() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.EventuallyWithTf(t, func(t *assert.CollectT) {
				m.Lock()
				defer m.Unlock()
				assert.Equal(t, tt.expectedCalls, calls)
			}, 5*time.Second, 200*time.Millisecond, "expected calls %v, but got %v", tt.expectedCalls, calls)
		})
	}
}

func Test_commandShutdown_Shutdown(t *testing.T) {
	tests := []struct {
		name          string
		run           func(name string, arg ...string) error
		wantErr       bool
		expectedCalls [][]string
	}{
		{
			name: "should report error when dry-run fails",
			run: func(name string, arg ...string) error {
				return errors.New("test error")
			},
			wantErr: true,
			expectedCalls: [][]string{
				{"shutdown", "-k", "--no-wall", "now"},
				{"shutdown", "-c"},
			},
		},
		{
			name: "should not report error when dry-run succeeds",
			run: func(name string, arg ...string) error {
				return nil
			},
			wantErr: false,
			expectedCalls: [][]string{
				{"shutdown", "-k", "--no-wall", "now"},
				{"shutdown", "-h", "now"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m sync.Mutex
			calls := make([][]string, 0)

			c := &commandShutdown{
				run: func(ctx context.Context, name string, arg ...string) error {
					m.Lock()
					defer m.Unlock()
					calls = append(calls, append([]string{name}, arg...))
					return tt.run(name, arg...)
				},
			}

			if err := c.Shutdown(); (err != nil) != tt.wantErr {
				t.Errorf("Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.EventuallyWithTf(t, func(t *assert.CollectT) {
				m.Lock()
				defer m.Unlock()
				assert.Equal(t, tt.expectedCalls, calls)
			}, 5*time.Second, 200*time.Millisecond, "expected calls %v, but got %v", tt.expectedCalls, calls)
		})
	}
}
