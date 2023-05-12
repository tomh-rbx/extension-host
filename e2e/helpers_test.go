package e2e

import (
  "errors"
  "testing"
)

var (
  counter = 0
)

func Test_awaitUntilAssertedError(t *testing.T) {
	type args struct {
		t       *testing.T
		f       func() error
		message string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test_awaitUntilAssertedError_ok",
			args: args{
				t: t,
				f: func() error {
					return errors.New("error")
				},
				message: "",
			},
		},{
			name: "Test_awaitUntilAssertedError_retry",
			args: args{
				t: t,
				f: func() error {
          counter++
          if counter == 2 {
					  return errors.New("error")
          }
          return nil
				},
				message: "",
			},
		},
	}
    t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
      awaitUntilAssertedError(tt.args.t, tt.args.f, tt.args.message)
		})
	}
}
