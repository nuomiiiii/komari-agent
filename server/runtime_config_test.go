package server

import (
	"testing"

	v2 "github.com/komari-monitor/komari-agent/protocol/v2"
	"github.com/komari-monitor/komari-agent/runtimeconfig"
)

func TestProcessBasicInfoResponseAppliesDisabledConfig(t *testing.T) {
	previous := runtimeconfig.MonthRotateDay()
	t.Cleanup(func() { runtimeconfig.SetMonthRotateDay(previous) })
	runtimeconfig.SetMonthRotateDay(26)

	body := []byte(`{"jsonrpc":"2.0","id":"test","result":{"status":"success","config":{"month_rotate":0}}}`)
	if err := processBasicInfoResponse(body, 2); err != nil {
		t.Fatalf("processBasicInfoResponse() error = %v", err)
	}
	if got := runtimeconfig.MonthRotateDay(); got != 0 {
		t.Fatalf("MonthRotateDay() = %d, want 0", got)
	}
}

func TestApplyRuntimeConfigRejectsInvalidDay(t *testing.T) {
	if err := applyRuntimeConfig(v2.ConfigParams{MonthRotate: 32}); err == nil {
		t.Fatal("applyRuntimeConfig() expected an error")
	}
}
