package comfyui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

const realRegistry = "../../comfyui/workflows.toml"

func loadReal(t *testing.T) *Registry {
	t.Helper()
	reg, err := Load(realRegistry, map[string]config.ComfyProfile{
		"fast":    {Workflow: "lemon_flux_schnell"},
		"general": {Workflow: "lemon_sdxl"},
	}, "general")
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func input(t *testing.T, graph map[string]any, ref string) any {
	t.Helper()
	b, err := parseBinding(ref, "test")
	if err != nil {
		t.Fatal(err)
	}
	v, err := lookupInput(graph, b)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAllCommittedWorkflowsLoad(t *testing.T) {
	reg := loadReal(t)
	if got := len(reg.Workflows()); got != 30 {
		t.Errorf("got %d workflows, want 30", got)
	}
	for _, w := range reg.Workflows() {
		if w.Unavailable != "" {
			t.Errorf("workflow %q unavailable: %s", w.ID, w.Unavailable)
		}
		if strings.Contains(w.ID, "_llm") {
			t.Errorf("prompt-rewriting workflow %q should not be registered", w.ID)
		}
		if len(w.Hash) != 12 {
			t.Errorf("workflow %q hash = %q", w.ID, w.Hash)
		}
	}
	// Every registered workflow must build for every aspect ratio.
	for _, w := range reg.Workflows() {
		for _, ar := range AspectRatios {
			reg.workflows[w.ID].Exposed = true
			if _, err := reg.Prepare(Request{Prompt: "a lemon", WorkflowID: w.ID, AspectRatio: ar}); err != nil {
				t.Errorf("%s/%s: %v", w.ID, ar, err)
			}
		}
	}
}

func TestSelection(t *testing.T) {
	reg := loadReal(t)
	tests := []struct {
		name         string
		req          Request
		wantWorkflow string
		wantProfile  string
		wantErr      string
	}{
		{"default profile", Request{}, "lemon_sdxl", "general", ""},
		{"fast profile", Request{Profile: "fast"}, "lemon_flux_schnell", "fast", ""},
		{"exact workflow", Request{WorkflowID: "lemon_sdxl"}, "lemon_sdxl", "", ""},
		{"both selectors", Request{Profile: "fast", WorkflowID: "lemon_sdxl"}, "", "", "not both"},
		{"unknown workflow", Request{WorkflowID: "nope"}, "", "", `unknown workflow_id "nope"`},
		{"path-like workflow", Request{WorkflowID: "../lemon.toml"}, "", "", "unknown workflow_id"},
		{"registered but not exposed", Request{WorkflowID: "juggernaut_xi"}, "", "", "unknown workflow_id"},
		{"unknown profile", Request{Profile: "best"}, "", "", `unknown profile "best"`},
		{"bad aspect", Request{AspectRatio: "wide"}, "", "", "invalid aspect_ratio"},
		{"negative seed", Request{Seed: -1}, "", "", "seed must be"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.req.Prompt = "a lemon"
			job, err := reg.Prepare(tt.req)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if job.Workflow.ID != tt.wantWorkflow || job.Profile != tt.wantProfile {
				t.Errorf("got workflow %q profile %q, want %q %q", job.Workflow.ID, job.Profile, tt.wantWorkflow, tt.wantProfile)
			}
		})
	}
	if _, err := reg.Prepare(Request{Prompt: "  "}); err == nil {
		t.Error("blank prompt accepted")
	}
}

func TestPrepareBindsInputs(t *testing.T) {
	reg := loadReal(t)

	job, err := reg.Prepare(Request{Prompt: "a lemon", Profile: "general", Seed: 42, AspectRatio: "landscape"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Seed != 42 || input(t, job.Graph, "5.seed") != int64(42) {
		t.Errorf("fixed seed not passed through: job=%d graph=%v", job.Seed, input(t, job.Graph, "5.seed"))
	}
	if job.Width != 1216 || job.Height != 832 || input(t, job.Graph, "4.width") != 1216 || input(t, job.Graph, "4.height") != 832 {
		t.Errorf("landscape size not bound: %dx%d", job.Width, job.Height)
	}
	if input(t, job.Graph, "2.text") != "a lemon" {
		t.Errorf("prompt not bound: %v", input(t, job.Graph, "2.text"))
	}

	job, err = reg.Prepare(Request{Prompt: "a lemon"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Seed <= 0 || job.Seed > maxRandomSeed || input(t, job.Graph, "5.seed") != job.Seed {
		t.Errorf("random seed %d not generated and bound", job.Seed)
	}

	// Multiple dimension targets, and text around the placeholder is kept.
	reg.workflows["flux_dev"].Exposed = true
	reg.workflows["pony_realism"].Exposed = true
	job, err = reg.Prepare(Request{Prompt: "a lemon", WorkflowID: "flux_dev", AspectRatio: "portrait"})
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"27.width", "30.width"} {
		if input(t, job.Graph, ref) != 832 {
			t.Errorf("%s = %v, want 832", ref, input(t, job.Graph, ref))
		}
	}
	job, err = reg.Prepare(Request{Prompt: "a lemon", WorkflowID: "pony_realism"})
	if err != nil {
		t.Fatal(err)
	}
	if got := input(t, job.Graph, "6.text"); got != "score_9, score_8_up, score_7_up, a lemon" {
		t.Errorf("prompt template = %q", got)
	}
}

func TestUntouchedNumbersSurviveVerbatim(t *testing.T) {
	reg := loadReal(t)
	reg.workflows["qwen_image"].Exposed = true
	job, err := reg.Prepare(Request{Prompt: "a lemon", WorkflowID: "qwen_image"})
	if err != nil {
		t.Fatal(err)
	}
	out, _ := json.Marshal(job.Graph)
	if !strings.Contains(string(out), `"shift":3.1000000000000005`) {
		t.Error("float input was not resubmitted verbatim")
	}
}

func TestNegativePrompt(t *testing.T) {
	reg := loadReal(t)
	reg.workflows["SDXL"].Exposed = true

	job, err := reg.Prepare(Request{Prompt: "a sign", WorkflowID: "SDXL"})
	if err != nil {
		t.Fatal(err)
	}
	def := "lowres, blurry, jpeg artifacts, watermark, signature, deformed, extra fingers"
	if job.NegativePrompt == nil || *job.NegativePrompt != def || input(t, job.Graph, "7.text") != def {
		t.Errorf("default negative not reported/kept: %v", job.NegativePrompt)
	}

	override := "lowres, blurry"
	job, err = reg.Prepare(Request{Prompt: "a sign", WorkflowID: "SDXL", NegativePrompt: &override})
	if err != nil {
		t.Fatal(err)
	}
	if *job.NegativePrompt != override || input(t, job.Graph, "7.text") != override {
		t.Errorf("override not applied: %v", input(t, job.Graph, "7.text"))
	}

	empty := ""
	job, err = reg.Prepare(Request{Prompt: "a sign", WorkflowID: "SDXL", NegativePrompt: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if input(t, job.Graph, "7.text") != "" {
		t.Error("empty override should clear the negative")
	}

	job, err = reg.Prepare(Request{Prompt: "a sign", Profile: "fast", NegativePrompt: &override})
	if err != nil {
		t.Fatal(err)
	}
	if job.NegativePrompt != nil || len(job.Warnings) != 1 {
		t.Errorf("workflow without negative: negative=%v warnings=%v", job.NegativePrompt, job.Warnings)
	}
}

func TestConcurrentPrepareDoesNotShareGraphs(t *testing.T) {
	reg := loadReal(t)
	var wg sync.WaitGroup
	errs := make(chan error, 50)
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			prompt := fmt.Sprintf("prompt %d", i)
			job, err := reg.Prepare(Request{Prompt: prompt, Seed: int64(i + 1)})
			if err != nil {
				errs <- err
				return
			}
			b, _ := parseBinding("2.text", "")
			v, _ := lookupInput(job.Graph, b)
			if v != prompt {
				errs <- fmt.Errorf("request %d got prompt %v", i, v)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if !strings.Contains(string(reg.workflows["lemon_sdxl"].raw), PromptPlaceholder) {
		t.Error("stored workflow was mutated")
	}
}

func writeRegistry(t *testing.T, toml string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "workflows.toml")
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const tinyGraph = `{
  "1": {"class_type": "CLIPTextEncode", "inputs": {"text": "%PROMPT%"}},
  "2": {"class_type": "EmptyLatentImage", "inputs": {"width": 1024, "height": ["9", 0]}},
  "3": {"class_type": "KSampler", "inputs": {"seed": 0}},
  "4": {"class_type": "SaveImage", "inputs": {}}
}`

const tinyEntry = `
[[workflow]]
id     = "%s"
file   = "g.json"
prompt = "1.text"
seed   = ["3.seed"]
width  = ["2.width"]
height = ["%s"]
output = "4"
sizes  = { square = [512, 512], landscape = [768, 512], portrait = [512, 768] }
`

func TestBrokenWorkflowIsUnavailableNotFatal(t *testing.T) {
	// height is linked to another node, so it cannot be bound.
	path := writeRegistry(t, fmt.Sprintf(tinyEntry, "broken", "2.height"), map[string]string{"g.json": tinyGraph})
	reg, err := Load(path, map[string]config.ComfyProfile{"general": {Workflow: "broken"}}, "general")
	if err != nil {
		t.Fatal(err)
	}
	w := reg.Workflows()[0]
	if !strings.Contains(w.Unavailable, "not a literal number") {
		t.Errorf("Unavailable = %q", w.Unavailable)
	}
	if _, err := reg.Prepare(Request{Prompt: "x"}); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Errorf("Prepare on unavailable workflow: %v", err)
	}
}

func TestLoadErrors(t *testing.T) {
	path := writeRegistry(t, fmt.Sprintf(tinyEntry, "ok", "2.width"), map[string]string{"g.json": tinyGraph})
	if _, err := Load(path, map[string]config.ComfyProfile{"general": {Workflow: "typo"}}, "general"); err == nil {
		t.Error("profile pointing at unknown workflow should fail to load")
	}

	dup := fmt.Sprintf(tinyEntry, "Same", "2.width") + fmt.Sprintf(tinyEntry, "same", "2.width")
	path = writeRegistry(t, dup, map[string]string{"g.json": tinyGraph})
	if _, err := Load(path, nil, ""); err == nil {
		t.Error("ids differing only by case should fail to load")
	}

	path = writeRegistry(t, fmt.Sprintf(tinyEntry, "ok", "2.width")+"bogus = 1\n", map[string]string{"g.json": tinyGraph})
	if _, err := Load(path, nil, ""); err == nil {
		t.Error("unknown registry keys should fail to load")
	}
}

func TestPromptPlaceholderMustAppearOnce(t *testing.T) {
	graph := strings.Replace(tinyGraph, `"%PROMPT%"`, `"no placeholder"`, 1)
	path := writeRegistry(t, fmt.Sprintf(tinyEntry, "w", "2.width"), map[string]string{"g.json": graph})
	reg, err := Load(path, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reg.Workflows()[0].Unavailable, "exactly once") {
		t.Errorf("Unavailable = %q", reg.Workflows()[0].Unavailable)
	}
}
