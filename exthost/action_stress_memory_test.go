package exthost

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-host/exthost/resources"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
	"testing"
)

func TestActionMemory_Prepare(t *testing.T) {
  osHostname = func() (string, error) {
    return "myhostname", nil
  }
	tests := []struct {
		name        string
		requestBody action_kit_api.PrepareActionRequestBody
		wantedError error
		wantedState *resources.StressActionState
	}{
		{
			name: "Should return config",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":     "prepare",
					"duration":   "1000",
					"percentage": "50",
				},
				ExecutionId: uuid.New(),
        Target: extutil.Ptr(action_kit_api.Target{
          Attributes: map[string][]string{
            "host.hostname":    {"myhostname"},
          },
        }),
			},

			wantedState: &resources.StressActionState{
				StressNGArgs: []string{"--vm", "1", "--vm-hang", "0", "--timeout", "1", "--vm-bytes"},
				Pid:          0,
			},
		}, {
			name: "Should return error too low duration",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":     "prepare",
					"duration":   "500",
					"percentage": "50",
				},
				ExecutionId: uuid.New(),
        Target: extutil.Ptr(action_kit_api.Target{
          Attributes: map[string][]string{
            "host.hostname":    {"myhostname"},
          },
        }),
			},

			wantedError: extutil.Ptr(extension_kit.ToError("Duration must be greater / equal than 1s", nil)),
		},
	}
	action := NewStressMemoryAction()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//Given
			state := resources.StressActionState{}
			request := tt.requestBody
			//When
			result, err := action.Prepare(context.Background(), &state, request)

			//Then
			if tt.wantedError != nil && err != nil {
				assert.EqualError(t, err, tt.wantedError.Error())
			} else if tt.wantedError != nil {
				assert.Equal(t, result.Error.Title, tt.wantedError.Error())
			}
			if tt.wantedState != nil {
				assert.NoError(t, err)
				assert.True(t, slices.Contains(state.StressNGArgs, tt.wantedState.StressNGArgs[0]))
				assert.True(t, slices.Contains(state.StressNGArgs, tt.wantedState.StressNGArgs[1]))
				assert.True(t, slices.Contains(state.StressNGArgs, tt.wantedState.StressNGArgs[2]))
				assert.True(t, slices.Contains(state.StressNGArgs, tt.wantedState.StressNGArgs[3]))
				assert.True(t, slices.Contains(state.StressNGArgs, tt.wantedState.StressNGArgs[4]))
				assert.True(t, slices.Contains(state.StressNGArgs, tt.wantedState.StressNGArgs[5]))
				assert.True(t, slices.Contains(state.StressNGArgs, tt.wantedState.StressNGArgs[6]))
				assert.Equal(t, tt.wantedState.Pid, state.Pid)
			}
		})
	}
}
