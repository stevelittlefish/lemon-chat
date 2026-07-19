package server

import (
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

func TestToolRegistryIsSelfContained(t *testing.T) {
	orders := make(map[int]string)
	for id, tool := range toolRegistry {
		if len(tool.Members) > 0 {
			if tool.Executor != nil || tool.Configure != nil || tool.Function.Name != "" {
				t.Errorf("compound tool %q has an executor, configurer, or model definition", id)
			}
			for _, memberID := range tool.Members {
				member, ok := toolRegistry[memberID]
				if !ok {
					t.Errorf("compound tool %q references unknown member %q", id, memberID)
					continue
				}
				if member.Function.Name != memberID {
					t.Errorf("compound tool %q member %q has function name %q", id, memberID, member.Function.Name)
				}
			}
		} else {
			if tool.Function.Name != id {
				t.Errorf("tool %q has function name %q", id, tool.Function.Name)
			}
			if tool.Executor == nil && tool.Configure == nil {
				t.Errorf("tool %q has no executor or configuration hook", id)
			}
		}

		if tool.DisplayName == "" {
			continue
		}
		if tool.Group == "" {
			t.Errorf("selectable tool %q has no frontend group", id)
		}
		if tool.Order <= 0 {
			t.Errorf("selectable tool %q has no positive order", id)
		} else if other, exists := orders[tool.Order]; exists {
			t.Errorf("selectable tools %q and %q share order %d", other, id, tool.Order)
		} else {
			orders[tool.Order] = id
		}
	}
}

func TestInitToolsDerivesExecutorsAndMetadata(t *testing.T) {
	defer InitTools(&config.Config{})
	cfg := &config.Config{}
	InitTools(cfg)

	visible := 0
	previousOrder := 0
	for _, meta := range allTools {
		tool := toolRegistry[meta.ID]
		visible++
		if meta.DisplayName != tool.DisplayName || meta.Group != tool.Group {
			t.Errorf("metadata for %q was not derived from registry", meta.ID)
		}
		if tool.Order <= previousOrder {
			t.Errorf("tool metadata is not ordered: %q has order %d after %d", meta.ID, tool.Order, previousOrder)
		}
		previousOrder = tool.Order
	}

	registryVisible := 0
	for _, tool := range toolRegistry {
		if tool.DisplayName != "" {
			registryVisible++
		}
	}
	if visible != registryVisible {
		t.Fatalf("got %d frontend tools, registry declares %d", visible, registryVisible)
	}
	if _, ok := executors["get_time"]; !ok {
		t.Error("static executor was not derived from registry")
	}
	if _, ok := executors["generate_image_sdxl"]; ok {
		t.Error("unconfigured SDXL executor should not be installed")
	}

	cfg.ComfyUI = config.ComfyUI{URL: "http://comfy.test", SDXLWorkflow: "sdxl.json", FluxWorkflow: "flux.json"}
	InitTools(cfg)
	if _, ok := executors["generate_image_sdxl"]; !ok {
		t.Error("configured SDXL executor was not installed")
	}
	if _, ok := executors["generate_image_flux"]; !ok {
		t.Error("configured Flux executor was not installed")
	}
}

func TestWorldStateCompoundIncludesClear(t *testing.T) {
	defs := ToolDefsForCharacter([]string{"world_state"})
	if len(defs) != len(toolRegistry["world_state"].Members) {
		t.Fatalf("got %d definitions, want %d", len(defs), len(toolRegistry["world_state"].Members))
	}
	for _, def := range defs {
		if def.Function.Name == "state_clear" {
			return
		}
	}
	t.Error("world_state did not expand to state_clear")
}
