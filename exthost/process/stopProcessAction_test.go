package stopprocess

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestActionStopProcess_Prepare(t *testing.T) {

	tests := []struct {
		name        string
		requestBody action_kit_api.PrepareActionRequestBody
		wantedError error
		wantedState *ActionState
	}{
		{
			name: "Should return config",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":   "prepare",
					"duration": "10000",
					"delay":    "1000",
					"graceful": "true",
					"process":  "tail",
				},
				ExecutionId: uuid.New(),
			},

			wantedState: &ActionState{
				ProcessOrPid: "tail",
				Graceful:     true,
				Duration:     10 * time.Second,
				Delay:        1 * time.Second,
			},
		}, {
			name: "Should return error too low duration",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":   "prepare",
					"duration": "0",
					"delay":    "1000",
					"graceful": "true",
					"process":  "tail",
				},
				ExecutionId: uuid.New(),
			},

			wantedError: extutil.Ptr(extension_kit.ToError("Duration is required", nil)),
		},
	}
	action := NewStopProcessAction()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//Given
			state := ActionState{}
			request := tt.requestBody
			now := time.Now()

			//When
			result, err := action.Prepare(context.Background(), &state, request)

			//Then
			if tt.wantedError != nil && err != nil {
				assert.EqualError(t, err, tt.wantedError.Error())
			} else if tt.wantedError != nil && result != nil {
				assert.Equal(t, result.Error.Title, tt.wantedError.Error())
			}
			if tt.wantedState != nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantedState.ProcessOrPid, state.ProcessOrPid)
				assert.Equal(t, tt.wantedState.Graceful, state.Graceful)
				assert.Equal(t, tt.wantedState.Delay, state.Delay)
				deadline := now.Add(state.Duration * time.Second)
				assert.GreaterOrEqual(t, deadline.Unix(), state.Deadline.Unix())
			}
		})
	}
}
