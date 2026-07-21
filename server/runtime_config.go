package server

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/komari-monitor/komari-agent/monitoring/netstatic"
	monitoring "github.com/komari-monitor/komari-agent/monitoring/unit"
	v2 "github.com/komari-monitor/komari-agent/protocol/v2"
	"github.com/komari-monitor/komari-agent/runtimeconfig"
)

type runtimeConfigEnvelope struct {
	Config             *v2.ConfigParams `json:"config,omitempty"`
	RequestConfigState bool             `json:"request_config_state,omitempty"`
}

func processBasicInfoResponse(body []byte, protocolVersion int) error {
	var envelope runtimeConfigEnvelope
	if protocolVersion >= 2 {
		response, err := parseV2Response(body)
		if err != nil {
			return err
		}
		if response.Result == nil {
			return nil
		}
		if err := v2.BindResult(response.Result, &envelope); err != nil {
			return fmt.Errorf("failed to parse runtime config: %w", err)
		}
	} else if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("failed to parse runtime config: %w", err)
	}
	if envelope.Config == nil {
		if envelope.RequestConfigState {
			return tryUploadDataWithProtocol(map[string]interface{}{
				"month_rotate": runtimeconfig.MonthRotateDay(),
			}, protocolVersion)
		}
		return nil
	}
	return applyRuntimeConfig(*envelope.Config)
}

func applyRuntimeConfig(config v2.ConfigParams) error {
	if config.MonthRotate < 0 || config.MonthRotate > 31 {
		return fmt.Errorf("month_rotate must be 0 or a day from 1 to 31")
	}
	current := runtimeconfig.MonthRotateDay()
	if current == config.MonthRotate {
		return nil
	}
	if config.MonthRotate == 0 {
		if err := netstatic.Stop(); err != nil {
			return fmt.Errorf("stop network statistics: %w", err)
		}
		runtimeconfig.SetMonthRotateDay(0)
		log.Println("Disabled monthly network traffic reset from Komari config")
		return nil
	}

	if err := netstatic.StartOrContinue(); err != nil {
		return fmt.Errorf("start network statistics: %w", err)
	}
	nics, err := monitoring.InterfaceList()
	if err != nil {
		return fmt.Errorf("list network interfaces: %w", err)
	}
	if err := netstatic.SetNewConfig(netstatic.NetStaticConfig{Nics: nics}); err != nil {
		return fmt.Errorf("configure network statistics: %w", err)
	}
	runtimeconfig.SetMonthRotateDay(config.MonthRotate)
	log.Printf("Updated monthly network traffic reset day to %d from Komari config", config.MonthRotate)
	return nil
}
