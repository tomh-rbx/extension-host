package exthost

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
  "github.com/steadybit/extension-kit/extutil"
  "github.com/stretchr/testify/assert"
	"testing"
)

func TestActionShutdown_Prepare(t *testing.T) {
  osHostname = func() (string, error) {
    return "myhostname", nil
  }
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
					"action": "prepare",
					"reboot": "true",
				},
				ExecutionId: uuid.New(),
        Target: extutil.Ptr(action_kit_api.Target{
          Attributes: map[string][]string{
            "host.hostname":    {"myhostname"},
          },
        }),
			},

			wantedState: &ActionState{
				Reboot:         true,
				ShutdownMethod: SyscallOrSysrq,
			},
		},
	}
	action := NewShutdownAction()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//Given
			state := ActionState{}
			request := tt.requestBody

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
				assert.Equal(t, tt.wantedState.Reboot, state.Reboot)
			}
		})
	}
}
