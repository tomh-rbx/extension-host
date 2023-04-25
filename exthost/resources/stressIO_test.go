package resources

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestActionIO_Prepare(t *testing.T) {

	tests := []struct {
		name        string
		requestBody action_kit_api.PrepareActionRequestBody
		wantedError error
		wantedState *StressActionState
	}{
		{
			name: "Should return config",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":   "prepare",
					"duration": "1000",
					"workers":  "1",
				},
				ExecutionId: uuid.New(),
			},

			wantedState: &StressActionState{
				StressNGArgs: []string{"--io", "1", "--timeout", "1", "--aggressive"},
				Pid:          0,
			},
		}, {
			name: "Should return error too low duration",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":   "prepare",
					"duration": "500",
					"workers":  "1",
				},
				ExecutionId: uuid.New(),
			},

			wantedError: extutil.Ptr(extension_kit.ToError("Duration must be greater / equal than 1s", nil)),
		},
	}
	action := NewStressIOAction()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//Given
			state := StressActionState{}
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
				assert.Equal(t, tt.wantedState.StressNGArgs, state.StressNGArgs)
				assert.Equal(t, tt.wantedState.Pid, state.Pid)
			}
		})
	}
}
