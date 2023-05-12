package e2e

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var (
	testInterval = 1 * time.Second
	maxAttempts  = 3
)

func awaitUntilAssertedError(t *testing.T, f func() error, message string) {
	retry := Retry(t, maxAttempts, testInterval, func(r *R) {
		err := f()
		if err == nil {
			r.failed = true
			r.log.WriteString("Error is nil")
		}
	})
	require.True(t, retry, message)
}

func awaitUntilAssertedNoError(t *testing.T, f func() error, message string) {
	retry := Retry(t, maxAttempts, testInterval, func(r *R) {
		err := f()
		if err != nil {
			r.failed = true
			r.log.WriteString(err.Error())
		}
	})
	require.True(t, retry, message)
}

func awaitUntilAssertedNoErrorUrl(t *testing.T, f func(param string) error, param string, message string) {
	retry := Retry(t, maxAttempts, testInterval, func(r *R) {
		err := f(param)
		if err != nil {
			r.failed = true
			r.log.WriteString(err.Error())
		}
	})
	require.True(t, retry, message)
}

func awaitUntilAssertedErrorUrl(t *testing.T, f func(param string) error, param string, message string) {
	retry := Retry(t, maxAttempts, testInterval, func(r *R) {
		err := f(param)
		if err == nil {
			r.failed = true
			r.log.WriteString("Error is nil")
		}
	})
	require.True(t, retry, message)
}
