package resources

import (
  "github.com/google/uuid"
  "github.com/steadybit/action-kit/go/action_kit_api/v2"
  extension_kit "github.com/steadybit/extension-kit"
  "github.com/steadybit/extension-kit/extutil"
  "github.com/stretchr/testify/assert"
  "testing"
)

func TestAction_Prepare(t *testing.T) {

  tests := []struct {
    name        string
    requestBody action_kit_api.PrepareActionRequestBody
    wantedError error
    wantedState *StressCPUActionState
  }{
    {
      name: "Should return config",
      requestBody: action_kit_api.PrepareActionRequestBody{
        Config: map[string]interface{}{
          "action":   "prepare",
          "duration": "1000",
          "workers":  "1",
          "cpuLoad":  "50",
        },
        ExecutionId: uuid.New(),
      },

      wantedState: &StressCPUActionState{
        StressNGArgs: []string{"--cpu", "1", "--cpu-load", "50", "--timeout", "1"},
        Pid:          0,
      },
    }, {
      name: "Should return error too low duration",
      requestBody: action_kit_api.PrepareActionRequestBody{
        Config: map[string]interface{}{
          "action":   "prepare",
          "duration": "500",
          "workers":  "1",
          "cpuLoad":  "50",
        },
        ExecutionId: uuid.New(),
      },

      wantedError: extutil.Ptr(extension_kit.ToError("Duration must be greater / equal than 1s", nil)),
    },
  }
  action := NewStressCPUAction()
  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      //Given
      state := StressCPUActionState{}
      request := tt.requestBody
      //When
      result, err := action.Prepare(nil, &state, request)

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
