// Package comfyui holds the workflow registry behind the generate_image tool:
// which ComfyUI API-format graphs exist, which of their inputs the tool may set,
// and how a request is turned into a concrete graph.
//
// Bindings are explicit ("node_id.input_name" in the registry file) and checked
// against each graph at load time. Nothing is inferred from filenames or node
// types, and a workflow that fails its checks is kept but marked unavailable
// with a reason, so it can never be submitted half-configured.
package comfyui

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

// PromptPlaceholder marks where the prompt goes inside the bound prompt input.
// Text around it (quality tags, score prefixes) is part of the workflow recipe.
const PromptPlaceholder = "%PROMPT%"

const (
	defaultTimeout = 120 * time.Second
	minDimension   = 64
	maxDimension   = 4096
	// Random seeds stay below 2^53 so they survive a round trip through
	// JavaScript numbers in the frontend and tool-call history.
	maxRandomSeed = 1<<53 - 1
)

// AspectRatios are the values the tool accepts, in schema order.
var AspectRatios = []string{"square", "landscape", "portrait"}

type binding struct {
	Node  string
	Input string
}

func (b binding) String() string { return b.Node + "." + b.Input }

// Workflow is one registered graph. The parsed graph is never handed out;
// Prepare decodes a fresh copy from raw for every request.
type Workflow struct {
	ID          string
	File        string
	Description string
	Exposed     bool
	Timeout     time.Duration
	Hash        string // sha256 of the file, first 12 hex chars
	Output      string // node ID whose images are returned
	Sizes       map[string][2]int

	// Unavailable is empty when the workflow can be used, otherwise the reason
	// it cannot (missing file, bad binding, ...).
	Unavailable string

	DefaultNegative string // value of the bound negative input in the file

	prompt   binding
	negative *binding
	seed     []binding
	width    []binding
	height   []binding
	raw      []byte
}

// HasNegative reports whether the workflow has a negative prompt input.
func (w *Workflow) HasNegative() bool { return w.negative != nil }

// Profile is a resolved profile → workflow mapping.
type Profile struct {
	Name        string
	Description string
	Workflow    *Workflow
}

// Registry is immutable after Load and safe for concurrent use.
type Registry struct {
	workflows      map[string]*Workflow
	order          []string
	profiles       map[string]Profile
	DefaultProfile string
}

type fileEntry struct {
	ID             string           `toml:"id"`
	File           string           `toml:"file"`
	Description    string           `toml:"description"`
	Exposed        bool             `toml:"exposed"`
	Timeout        int              `toml:"timeout"`
	Prompt         string           `toml:"prompt"`
	NegativePrompt string           `toml:"negative_prompt"`
	Seed           []string         `toml:"seed"`
	Width          []string         `toml:"width"`
	Height         []string         `toml:"height"`
	Output         string           `toml:"output"`
	Sizes          map[string][]int `toml:"sizes"`
}

// Load reads the registry file and resolves profiles against it. Problems with
// the registry file itself or with a profile mapping are returned as errors;
// problems with an individual workflow mark only that workflow unavailable.
func Load(path string, profiles map[string]config.ComfyProfile, defaultProfile string) (*Registry, error) {
	var file struct {
		Workflow []fileEntry `toml:"workflow"`
	}
	meta, err := toml.DecodeFile(path, &file)
	if err != nil {
		return nil, fmt.Errorf("comfyui: read workflow registry %q: %w", path, err)
	}
	if keys := meta.Undecoded(); len(keys) > 0 {
		return nil, fmt.Errorf("comfyui: workflow registry %q has unknown keys: %v", path, keys)
	}

	r := &Registry{
		workflows:      make(map[string]*Workflow),
		profiles:       make(map[string]Profile),
		DefaultProfile: defaultProfile,
	}
	baseDir := filepath.Dir(path)
	lowerIDs := make(map[string]string)
	for _, e := range file.Workflow {
		if e.ID == "" {
			return nil, fmt.Errorf("comfyui: workflow registry %q has an entry with no id", path)
		}
		if other, ok := lowerIDs[strings.ToLower(e.ID)]; ok {
			return nil, fmt.Errorf("comfyui: workflow ids %q and %q differ only by case", other, e.ID)
		}
		lowerIDs[strings.ToLower(e.ID)] = e.ID
		w := loadWorkflow(baseDir, e)
		r.workflows[w.ID] = w
		r.order = append(r.order, w.ID)
	}

	for name, p := range profiles {
		w, ok := r.workflows[p.Workflow]
		if !ok {
			return nil, fmt.Errorf("comfyui: profile %q references unknown workflow %q", name, p.Workflow)
		}
		r.profiles[name] = Profile{Name: name, Description: p.Description, Workflow: w}
	}
	return r, nil
}

func loadWorkflow(baseDir string, e fileEntry) *Workflow {
	w := &Workflow{
		ID:          e.ID,
		File:        e.File,
		Description: e.Description,
		Exposed:     e.Exposed,
		Timeout:     defaultTimeout,
		Output:      e.Output,
	}
	if e.Timeout > 0 {
		w.Timeout = time.Duration(e.Timeout) * time.Second
	}
	if err := w.configure(baseDir, e); err != nil {
		w.Unavailable = err.Error()
	}
	return w
}

func (w *Workflow) configure(baseDir string, e fileEntry) error {
	if e.File == "" {
		return fmt.Errorf("no file configured")
	}
	if e.Timeout < 0 {
		return fmt.Errorf("timeout must be positive")
	}
	raw, err := os.ReadFile(filepath.Join(baseDir, e.File))
	if err != nil {
		return fmt.Errorf("cannot read workflow file: %w", err)
	}
	sum := sha256.Sum256(raw)
	w.Hash = hex.EncodeToString(sum[:])[:12]
	w.raw = raw

	graph, err := decodeGraph(raw)
	if err != nil {
		return fmt.Errorf("workflow file is not a valid ComfyUI API-format graph: %w", err)
	}

	if w.prompt, err = parseBinding(e.Prompt, "prompt"); err != nil {
		return err
	}
	tmpl, err := stringInput(graph, w.prompt)
	if err != nil {
		return err
	}
	if n := strings.Count(tmpl, PromptPlaceholder); n != 1 {
		return fmt.Errorf("prompt input %s must contain %s exactly once (found %d)", w.prompt, PromptPlaceholder, n)
	}

	if e.NegativePrompt != "" {
		b, err := parseBinding(e.NegativePrompt, "negative_prompt")
		if err != nil {
			return err
		}
		if w.DefaultNegative, err = stringInput(graph, b); err != nil {
			return err
		}
		w.negative = &b
	}

	for _, spec := range []struct {
		name   string
		values []string
		dst    *[]binding
	}{
		{"seed", e.Seed, &w.seed},
		{"width", e.Width, &w.width},
		{"height", e.Height, &w.height},
	} {
		if len(spec.values) == 0 {
			return fmt.Errorf("no %s binding configured", spec.name)
		}
		for _, v := range spec.values {
			b, err := parseBinding(v, spec.name)
			if err != nil {
				return err
			}
			if err := numberInput(graph, b); err != nil {
				return err
			}
			*spec.dst = append(*spec.dst, b)
		}
	}

	if e.Output == "" {
		return fmt.Errorf("no output node configured")
	}
	if _, ok := graph[e.Output]; !ok {
		return fmt.Errorf("output node %q does not exist", e.Output)
	}

	w.Sizes = make(map[string][2]int, len(AspectRatios))
	for _, ar := range AspectRatios {
		dims, ok := e.Sizes[ar]
		if !ok || len(dims) != 2 {
			return fmt.Errorf("sizes.%s must be [width, height]", ar)
		}
		for _, d := range dims {
			if d < minDimension || d > maxDimension || d%8 != 0 {
				return fmt.Errorf("sizes.%s: %d must be a multiple of 8 between %d and %d", ar, d, minDimension, maxDimension)
			}
		}
		w.Sizes[ar] = [2]int{dims[0], dims[1]}
	}
	for ar := range e.Sizes {
		if _, ok := w.Sizes[ar]; !ok {
			return fmt.Errorf("sizes has unknown aspect ratio %q (want %s)", ar, strings.Join(AspectRatios, ", "))
		}
	}
	return nil
}

// decodeGraph parses a workflow keeping numbers as json.Number, so values we
// don't touch (e.g. a shift of 3.1000000000000005) are resubmitted verbatim.
func decodeGraph(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var graph map[string]any
	if err := dec.Decode(&graph); err != nil {
		return nil, err
	}
	for id, n := range graph {
		node, ok := n.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("node %q is not an object", id)
		}
		if _, ok := node["class_type"].(string); !ok {
			return nil, fmt.Errorf("node %q has no class_type", id)
		}
		if _, ok := node["inputs"].(map[string]any); !ok {
			return nil, fmt.Errorf("node %q has no inputs object", id)
		}
	}
	return graph, nil
}

// parseBinding splits "node_id.input_name" at the first dot. Node IDs may
// contain colons (subgraph IDs like "60:19") but never dots.
func parseBinding(s, what string) (binding, error) {
	node, input, ok := strings.Cut(s, ".")
	if !ok || node == "" || input == "" {
		return binding{}, fmt.Errorf("%s binding %q must be \"node_id.input_name\"", what, s)
	}
	return binding{Node: node, Input: input}, nil
}

func lookupInput(graph map[string]any, b binding) (any, error) {
	n, ok := graph[b.Node]
	if !ok {
		return nil, fmt.Errorf("binding %s: node %q does not exist", b, b.Node)
	}
	inputs := n.(map[string]any)["inputs"].(map[string]any)
	v, ok := inputs[b.Input]
	if !ok {
		return nil, fmt.Errorf("binding %s: node %q has no input %q", b, b.Node, b.Input)
	}
	return v, nil
}

func stringInput(graph map[string]any, b binding) (string, error) {
	v, err := lookupInput(graph, b)
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("binding %s: input is not a literal string (is it linked to another node?)", b)
	}
	return s, nil
}

func numberInput(graph map[string]any, b binding) error {
	v, err := lookupInput(graph, b)
	if err != nil {
		return err
	}
	if _, ok := v.(json.Number); !ok {
		return fmt.Errorf("binding %s: input is not a literal number (is it linked to another node?)", b)
	}
	return nil
}

func setInput(graph map[string]any, b binding, v any) {
	graph[b.Node].(map[string]any)["inputs"].(map[string]any)[b.Input] = v
}

// Workflows returns every registered workflow in registry-file order.
func (r *Registry) Workflows() []*Workflow {
	out := make([]*Workflow, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.workflows[id])
	}
	return out
}

// Exposed returns the usable workflows the model may select by ID.
func (r *Registry) Exposed() []*Workflow {
	var out []*Workflow
	for _, w := range r.Workflows() {
		if w.Exposed && w.Unavailable == "" {
			out = append(out, w)
		}
	}
	return out
}

// Profiles returns the configured profiles sorted by name.
func (r *Registry) Profiles() []Profile {
	out := make([]Profile, 0, len(r.profiles))
	for _, p := range r.profiles {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// LogStatus writes one line per workflow and profile so an unavailable
// workflow is visible at startup rather than at first use.
func (r *Registry) LogStatus() {
	for _, w := range r.Workflows() {
		if w.Unavailable != "" {
			log.Printf("comfyui: workflow %q unavailable — %s", w.ID, w.Unavailable)
		}
	}
	for _, p := range r.Profiles() {
		status := "ok"
		if p.Workflow.Unavailable != "" {
			status = "UNAVAILABLE: " + p.Workflow.Unavailable
		}
		log.Printf("comfyui: profile %q → workflow %q (%s)", p.Name, p.Workflow.ID, status)
	}
	log.Printf("comfyui: %d workflow(s) registered, %d exposed, default profile %q", len(r.order), len(r.Exposed()), r.DefaultProfile)
}

// Request is a generate_image call after argument decoding.
type Request struct {
	Prompt         string
	Profile        string
	WorkflowID     string
	AspectRatio    string
	Seed           int64
	NegativePrompt *string // nil = use the workflow default
}

// Job is a fully resolved request: a private graph ready to submit plus the
// values that were actually used.
type Job struct {
	Workflow    *Workflow
	Profile     string // empty when selected by workflow_id
	Graph       map[string]any
	Seed        int64
	Width       int
	Height      int
	AspectRatio string
	// NegativePrompt is the negative actually sent, or nil when the workflow
	// has no negative input.
	NegativePrompt *string
	Warnings       []string
}

// Prepare validates a request, selects its workflow deterministically, and
// builds a private copy of the graph with the bound inputs set. It never falls
// back to a different workflow: any problem is returned as an error.
func (r *Registry) Prepare(req Request) (*Job, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	if req.Profile != "" && req.WorkflowID != "" {
		return nil, fmt.Errorf("pass either profile or workflow_id, not both")
	}

	job := &Job{}
	switch {
	case req.WorkflowID != "":
		w, ok := r.workflows[req.WorkflowID]
		if !ok || !w.Exposed {
			return nil, fmt.Errorf("unknown workflow_id %q — valid workflow IDs: %s", req.WorkflowID, r.exposedIDs())
		}
		job.Workflow = w
	default:
		name := req.Profile
		if name == "" {
			name = r.DefaultProfile
		}
		p, ok := r.profiles[name]
		if !ok {
			return nil, fmt.Errorf("unknown profile %q — valid profiles: %s", name, r.profileNames())
		}
		job.Workflow = p.Workflow
		job.Profile = p.Name
	}
	w := job.Workflow
	if w.Unavailable != "" {
		return nil, fmt.Errorf("workflow %q is unavailable: %s — tell the user image generation is misconfigured", w.ID, w.Unavailable)
	}

	job.AspectRatio = req.AspectRatio
	if job.AspectRatio == "" {
		job.AspectRatio = "square"
	}
	size, ok := w.Sizes[job.AspectRatio]
	if !ok {
		return nil, fmt.Errorf("invalid aspect_ratio %q — use one of: %s", req.AspectRatio, strings.Join(AspectRatios, ", "))
	}
	job.Width, job.Height = size[0], size[1]

	switch {
	case req.Seed < 0:
		return nil, fmt.Errorf("seed must be a positive integer, or omitted for a random seed")
	case req.Seed == 0:
		job.Seed = rand.Int63n(maxRandomSeed) + 1
	default:
		job.Seed = req.Seed
	}

	graph, err := decodeGraph(w.raw)
	if err != nil {
		return nil, fmt.Errorf("workflow %q could not be decoded: %w", w.ID, err)
	}
	tmpl := graph[w.prompt.Node].(map[string]any)["inputs"].(map[string]any)[w.prompt.Input].(string)
	setInput(graph, w.prompt, strings.Replace(tmpl, PromptPlaceholder, req.Prompt, 1))
	for _, b := range w.seed {
		setInput(graph, b, job.Seed)
	}
	for _, b := range w.width {
		setInput(graph, b, job.Width)
	}
	for _, b := range w.height {
		setInput(graph, b, job.Height)
	}
	if w.negative != nil {
		neg := w.DefaultNegative
		if req.NegativePrompt != nil {
			neg = *req.NegativePrompt
			setInput(graph, *w.negative, neg)
		}
		job.NegativePrompt = &neg
	} else if req.NegativePrompt != nil && strings.TrimSpace(*req.NegativePrompt) != "" {
		job.Warnings = append(job.Warnings, fmt.Sprintf("workflow %q does not use a negative prompt, so negative_prompt was ignored", w.ID))
	}
	job.Graph = graph
	return job, nil
}

func (r *Registry) exposedIDs() string {
	var ids []string
	for _, w := range r.Exposed() {
		ids = append(ids, w.ID)
	}
	if len(ids) == 0 {
		return "(none)"
	}
	return strings.Join(ids, ", ")
}

func (r *Registry) profileNames() string {
	var names []string
	for _, p := range r.Profiles() {
		names = append(names, p.Name)
	}
	return strings.Join(names, ", ")
}
