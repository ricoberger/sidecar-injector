package log

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDebug(t *testing.T) {
	require.NotPanics(t, func() {
		Debug("Test Debug Log")
	})
}

func TestInfo(t *testing.T) {
	require.NotPanics(t, func() {
		Info("Test Info Log")
	})
}

func TestWarn(t *testing.T) {
	require.NotPanics(t, func() {
		Warn("Test Warn Log")
	})
}

func TestError(t *testing.T) {
	require.NotPanics(t, func() {
		Error("Test Error Log")
	})
}

func TestPanic(t *testing.T) {
	require.Panics(t, func() {
		Panic("Test Panic Log")
	})
}
