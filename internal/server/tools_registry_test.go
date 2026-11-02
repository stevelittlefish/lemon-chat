package server

import (
	"reflect"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/comfyui"
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
	if _, ok := executors["generate_image"]; ok {
		t.Error("unconfigured image executor should not be installed")
	}
	if _, ok := toolRegistry["generate_image"].Function.Parameters.Properties["profile"].(map[string]any)["enum"]; ok {
		t.Error("unconfigured image tool should not list profiles")
	}

	cfg.ComfyUI = config.ComfyUI{
		URL:            "http://comfy.test",
		Workflows:      "../../comfyui/workflows.toml",
		DefaultProfile: "general",
		Profiles: map[string]config.ComfyProfile{
			"fast":    {Workflow: "lemon_flux_schnell"},
			"general": {Workflow: "lemon_sdxl"},
		},
	}
	InitTools(cfg)
	if _, ok := executors["generate_image"]; !ok {
		t.Error("configured image executor was not installed")
	}
	props := toolRegistry["generate_image"].Function.Parameters.Properties
	if got := props["profile"].(map[string]any)["enum"]; !reflect.DeepEqual(got, []any{"fast", "general", nil}) {
		t.Errorf("profile enum = %v", got)
	}
	// Which workflows are exposed is registry configuration, so derive it.
	reg, err := comfyui.Load(cfg.ComfyUI.Workflows, cfg.ComfyUI.Profiles, cfg.ComfyUI.DefaultProfile)
	if err != nil {
		t.Fatal(err)
	}
	var want []any
	for _, w := range reg.Exposed() {
		want = append(want, w.ID)
	}
	want = append(want, nil)
	if got := props["workflow_id"].(map[string]any)["enum"]; len(want) < 2 || !reflect.DeepEqual(got, want) {
		t.Errorf("workflow_id enum = %v, want %v", got, want)
	}

	// Re-initialising without ComfyUI must drop the configured schema again.
	InitTools(&config.Config{})
	if _, ok := toolRegistry["generate_image"].Function.Parameters.Properties["profile"].(map[string]any)["enum"]; ok {
		t.Error("schema override from previous config leaked into re-init")
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
