package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/comfyui"
	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// fakeComfy is a minimal ComfyUI: /prompt records the graph, /history reports
// an image on the expected output node (or an execution error), /view serves bytes.
type fakeComfy struct {
	mu        sync.Mutex
	submitted []map[string]any
	fail      string // exception message to report instead of an image
}

func (f *fakeComfy) handler(outputNode string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /prompt", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Prompt map[string]any `json:"prompt"`
		}
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		f.mu.Lock()
		f.submitted = append(f.submitted, body.Prompt)
		f.mu.Unlock()
		w.Write([]byte(`{"prompt_id":"p1"}`)) //nolint:errcheck
	})
	mux.HandleFunc("GET /history/p1", func(w http.ResponseWriter, r *http.Request) {
		if f.fail != "" {
			w.Write([]byte(`{"p1":{"outputs":{},"status":{"status_str":"error","completed":false,"messages":[["execution_start",{}],["execution_error",{"node_id":"4","node_type":"CheckpointLoaderSimple","exception_message":"` + f.fail + `"}]]}}}`)) //nolint:errcheck
			return
		}
		w.Write([]byte(`{"p1":{"outputs":{"` + outputNode + `":{"images":[{"filename":"x.png","subfolder":"","type":"output"}]}},"status":{"status_str":"success","completed":true}}}`)) //nolint:errcheck
	})
	mux.HandleFunc("GET /view", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PNGDATA")) //nolint:errcheck
	})
	return mux
}

func newImageTest(t *testing.T, fake *fakeComfy) (toolExecutor, ToolContext, *store.Store) {
	t.Helper()
	old := comfyPollInterval
	comfyPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { comfyPollInterval = old })

	reg, err := comfyui.Load("../../comfyui/workflows.toml", map[string]config.ComfyProfile{
		"fast":    {Workflow: "lemon_flux_schnell"},
		"general": {Workflow: "lemon_sdxl"},
	}, "general")
	if err != nil {
		t.Fatal(err)
	}
	// Tests use the default profile (lemon_sdxl), which saves from node 7.
	srv := httptest.NewServer(fake.handler("7"))
	t.Cleanup(srv.Close)

	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	user, err := st.CreateUser("alice", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	model := "m"
	conv, err := st.CreateConversation(user.ID, nil, &model, nil)
	if err != nil {
		t.Fatal(err)
	}
	tctx := ToolContext{
		ToolCallID:     "call_1",
		UserID:         user.ID,
		ConversationID: conv.ID,
		Store:          st,
		DataDir:        t.TempDir(),
		Hub:            NewHub(st),
	}
	return makeImageExecutor(srv.URL, reg), tctx, st
}

func TestGenerateImageSync(t *testing.T) {
	fake := &fakeComfy{}
	exec, tctx, st := newImageTest(t, fake)

	out, err := exec(`{"prompt":"a lemon","seed":42,"aspect_ratio":"portrait","async":false,"background":true}`, tctx)
	if err != nil {
		t.Fatal(err)
	}
	var res imageResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}
	if res.WorkflowID != "lemon_sdxl" || res.Profile != "general" || res.Seed != 42 ||
		res.Width != 832 || res.Height != 1216 || len(res.WorkflowHash) != 12 || res.Status != "" || !res.Background {
		t.Errorf("unexpected result: %s", out)
	}
	if res.NegativePrompt == nil || *res.NegativePrompt != "" {
		t.Errorf("lemon_sdxl default negative should be reported as empty: %s", out)
	}

	graph := fake.submitted[0]
	inputs := func(node string) map[string]any { return graph[node].(map[string]any)["inputs"].(map[string]any) }
	if inputs("2")["text"] != "a lemon" || inputs("5")["seed"] != float64(42) || inputs("4")["width"] != float64(832) {
		t.Errorf("submitted graph not bound: %v", graph)
	}

	att, err := st.GetAttachment(res.AttachmentID)
	if err != nil {
		t.Fatal(err)
	}
	if att.Status != "ready" {
		t.Errorf("attachment status = %q", att.Status)
	}
	if data, err := os.ReadFile(filepath.Join(tctx.DataDir, att.DiskPath)); err != nil || string(data) != "PNGDATA" {
		t.Errorf("image not saved: %v %q", err, data)
	}
	conv, err := st.GetConversation(tctx.ConversationID, tctx.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if conv.BackgroundAttachmentID == nil || *conv.BackgroundAttachmentID != att.ID {
		t.Error("background was not set")
	}
}

func TestGenerateImageAsync(t *testing.T) {
	fake := &fakeComfy{}
	exec, tctx, st := newImageTest(t, fake)

	out, err := exec(`{"prompt":"a lemon"}`, tctx)
	if err != nil {
		t.Fatal(err)
	}
	var res imageResult
	json.Unmarshal([]byte(out), &res) //nolint:errcheck
	if res.Status != "pending" || res.Seed <= 0 || res.Width != 1024 {
		t.Fatalf("unexpected pending result: %s", out)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		att, err := st.GetAttachment(res.AttachmentID)
		if err != nil {
			t.Fatal(err)
		}
		if att.Status == "ready" {
			break
		}
		if att.Status == "error" || time.Now().After(deadline) {
			t.Fatalf("attachment status = %q error = %q", att.Status, att.Error)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := fake.submitted[0]["5"].(map[string]any)["inputs"].(map[string]any)["seed"]; got != float64(res.Seed) {
		t.Errorf("returned seed %d does not match submitted seed %v", res.Seed, got)
	}
}

// Models that fill in every parameter send null for the ones they don't want.
func TestGenerateImageAcceptsNulls(t *testing.T) {
	fake := &fakeComfy{}
	exec, tctx, _ := newImageTest(t, fake)

	out, err := exec(`{"prompt":"a lemon","profile":null,"workflow_id":"lemon_sdxl","negative_prompt":null,"seed":null,"async":false}`, tctx)
	if err != nil {
		t.Fatal(err)
	}
	var res imageResult
	json.Unmarshal([]byte(out), &res) //nolint:errcheck
	if res.WorkflowID != "lemon_sdxl" || res.Seed <= 0 || res.NegativePrompt == nil {
		t.Errorf("nulls not treated as omitted: %s", out)
	}
	if _, err := exec(`{"prompt":"a lemon","profile":"general","workflow_id":null,"async":false}`, tctx); err != nil {
		t.Errorf("profile with null workflow_id: %v", err)
	}
}

func TestGenerateImageReportsComfyError(t *testing.T) {
	fake := &fakeComfy{fail: "ckpt not found"}
	exec, tctx, _ := newImageTest(t, fake)

	_, err := exec(`{"prompt":"a lemon","async":false}`, tctx)
	if err == nil || !strings.Contains(err.Error(), "ckpt not found") || !strings.Contains(err.Error(), "CheckpointLoaderSimple") {
		t.Fatalf("err = %v", err)
	}
}

func TestGenerateImageValidatesBeforeSubmitting(t *testing.T) {
	fake := &fakeComfy{}
	exec, tctx, _ := newImageTest(t, fake)

	for _, args := range []string{
		`{"prompt":"a lemon","profile":"fast","workflow_id":"lemon_sdxl"}`,
		`{"prompt":"a lemon","workflow_id":"juggernaut_xi"}`,
		`{"prompt":"a lemon","workflow_id":"../../lemon.toml"}`,
		`{"prompt":"a lemon","aspect_ratio":"panorama"}`,
		`{"prompt":""}`,
	} {
		if _, err := exec(args, tctx); err == nil {
			t.Errorf("%s: expected an error", args)
		}
	}
	if len(fake.submitted) != 0 {
		t.Errorf("invalid requests reached ComfyUI: %d", len(fake.submitted))
	}
}
