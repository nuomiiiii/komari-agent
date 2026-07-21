package cmd

import (
	"testing"

	pkg_flags "github.com/komari-monitor/komari-agent/cmd/flags"
)

func validRuntimeConfig() *pkg_flags.Config {
	return &pkg_flags.Config{
		Interval:           3,
		ReconnectInterval:  5,
		InfoReportInterval: 5,
		MaxRetries:         3,
		ProtocolVersion:    2,
	}
}

func TestValidateRuntimeConfig(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*pkg_flags.Config)
		valid  bool
	}{
		{name: "defaults", valid: true},
		{name: "zero interval", mutate: func(c *pkg_flags.Config) { c.Interval = 0 }},
		{name: "zero reconnect interval", mutate: func(c *pkg_flags.Config) { c.ReconnectInterval = 0 }},
		{name: "zero info interval", mutate: func(c *pkg_flags.Config) { c.InfoReportInterval = 0 }},
		{name: "negative retries", mutate: func(c *pkg_flags.Config) { c.MaxRetries = -1 }},
		{name: "invalid month day", mutate: func(c *pkg_flags.Config) { c.MonthRotate = 32 }},
		{name: "invalid protocol", mutate: func(c *pkg_flags.Config) { c.ProtocolVersion = 3 }},
		{name: "ipv4 preferred", mutate: func(c *pkg_flags.Config) { c.PreferIPVersion = "4" }, valid: true},
		{name: "invalid preferred IP", mutate: func(c *pkg_flags.Config) { c.PreferIPVersion = "auto" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validRuntimeConfig()
			if tt.mutate != nil {
				tt.mutate(config)
			}
			err := validateRuntimeConfig(config)
			if tt.valid && err != nil {
				t.Fatalf("validateRuntimeConfig() error = %v", err)
			}
			if !tt.valid && err == nil {
				t.Fatal("validateRuntimeConfig() expected an error")
			}
		})
	}
}
