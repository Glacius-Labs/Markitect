package projectexecutor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

func appServerConfig(config agentexec.Config) (codexappserver.Config, error) {
	if config.Transport != projectconfig.TransportCodexAppServer {
		return codexappserver.Config{}, fmt.Errorf("App Server adapter requires transport %q", projectconfig.TransportCodexAppServer)
	}
	settings, err := DecodeAppServerSettings(config.TransportConfig)
	if err != nil {
		return codexappserver.Config{}, err
	}
	return codexappserver.Config{
		Command: config.Command, ProviderVersion: config.ProviderVersion, Model: config.Model,
		ReasoningEffort: settings.ReasoningEffort, PermissionProfile: settings.PermissionProfile,
		WindowsSandboxBackend: settings.WindowsSandboxBackend,
		Helpers: codexappserver.HelperPolicy{Enabled: settings.Helpers.Enabled,
			MaxStartRequests: settings.Helpers.MaxStartRequests, MaxDepth: settings.Helpers.MaxDepth},
		Timeout: config.Timeout, MaxEventBytes: settings.MaxEventBytes,
	}, nil
}

func DecodeAppServerSettings(raw json.RawMessage) (projectconfig.AppServerSettings, error) {
	var settings projectconfig.AppServerSettings
	if len(raw) == 0 {
		return settings, errors.New("codex-app-server requires explicit transportConfig")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&settings); err != nil {
		return settings, fmt.Errorf("decode codex-app-server transportConfig: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return settings, errors.New("codex-app-server transportConfig must contain exactly one object")
	}
	return settings, nil
}

func validateNativeInvocationConfig(config agentexec.Config) error {
	adapterConfig, err := appServerConfig(config)
	if err != nil {
		return err
	}
	if config.Args != nil || config.WorkspaceMode != "" {
		return errors.New("App Server transport does not accept process args or workspace mode")
	}
	if len(config.ModelOptions) > 0 {
		var options map[string]json.RawMessage
		if json.Unmarshal(config.ModelOptions, &options) != nil || len(options) != 0 {
			return errors.New("App Server model options belong in explicit transport configuration")
		}
	}
	if adapterConfig.Timeout > time.Hour || adapterConfig.MaxEventBytes > 256<<20 {
		return errors.New("App Server time/event budget exceeds supported bounds")
	}
	return nil
}
