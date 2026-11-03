package llm

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestBuildResponsesBody(t *testing.T) {
	messages := []map[string]any{
		{"role": "system", "content": "Be terse."},
		{"role": "user", "content": "Hi"},
		{"role": "assistant", "content": "prior", "tool_calls": []any{
			map[string]any{"id": "call_1", "type": "function", "function": map[string]any{"name": "get_time", "arguments": "{}"}},
		}},
		{"role": "tool", "tool_call_id": "call_1", "content": "12:00"},
	}
	tools := []map[string]any{
		{"type": "function", "function": map[string]any{"name": "get_time", "description": "now", "parameters": map[string]any{"type": "object"}}},
	}
	raw, err := BuildResponsesBody("gpt-x", messages, tools, map[string]any{"max_tokens": 500, "temperature": 0.7}, true)
	if err != nil {
		t.Fatalf("BuildResponsesBody: %v", err)
	}
	var req map[string]any
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}
	if req["instructions"] != "Be terse." {
		t.Errorf("instructions = %v", req["instructions"])
	}
	if _, ok := req["temperature"]; ok {
		t.Error("temperature must be dropped for the Responses endpoint")
	}
	if _, ok := req["max_output_tokens"]; ok {
		t.Error("max_output_tokens must be dropped for the Responses endpoint")
	}
	if req["store"] != false || req["stream"] != true {
		t.Errorf("store/stream = %v/%v", req["store"], req["stream"])
	}
	input := req["input"].([]any)
	// user message, assistant message, function_call, function_call_output
	if len(input) != 4 {
		t.Fatalf("expected 4 input items, got %d: %v", len(input), input)
	}
	fc := input[2].(map[string]any)
	if fc["type"] != "function_call" || fc["call_id"] != "call_1" || fc["name"] != "get_time" {
		t.Errorf("function_call item wrong: %v", fc)
	}
	fco := input[3].(map[string]any)
	if fco["type"] != "function_call_output" || fco["call_id"] != "call_1" || fco["output"] != "12:00" {
		t.Errorf("function_call_output item wrong: %v", fco)
	}
	toolsOut := req["tools"].([]any)
	if len(toolsOut) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(toolsOut))
	}
	tool := toolsOut[0].(map[string]any)
	if tool["type"] != "function" || tool["name"] != "get_time" {
		t.Errorf("tool flattening wrong: %v", tool)
	}
	if req["tool_choice"] != "auto" {
		t.Errorf("tool_choice = %v", req["tool_choice"])
	}
}

// collectFrames reads chat-completions SSE frames from r into decoded maps.
func collectFrames(t *testing.T, r io.Reader) []map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read converted stream: %v", err)
	}
	var frames []map[string]any
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var f map[string]any
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &f) == nil {
			frames = append(frames, f)
		}
	}
	return frames
}

func sse(events ...string) string {
	var b strings.Builder
	for _, e := range events {
		b.WriteString("data: ")
		b.WriteString(e)
		b.WriteString("\n\n")
	}
	return b.String()
}

func TestResponsesToChatSSETextAndUsage(t *testing.T) {
	in := sse(
		`{"type":"response.created","response":{"id":"r1"}}`,
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"message"}}`,
		`{"type":"response.output_text.delta","output_index":0,"delta":"Hello"}`,
		`{"type":"response.output_text.delta","output_index":0,"delta":" world"}`,
		`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":10,"output_tokens":3,"total_tokens":13}}}`,
	)
	comp, err := readChatCompletionsStream(ResponsesToChatSSE(strings.NewReader(in)), nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if comp.Content != "Hello world" {
		t.Errorf("content = %q", comp.Content)
	}
	if comp.FinishReason != "stop" {
		t.Errorf("finish = %q", comp.FinishReason)
	}
	if comp.Usage == nil || comp.Usage.PromptTokens != 10 || comp.Usage.CompletionTokens != 3 {
		t.Errorf("usage = %+v", comp.Usage)
	}
}

func TestResponsesToChatSSEToolCalls(t *testing.T) {
	in := sse(
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"c1","name":"roll_dice"}}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"sides\":"}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"6}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"c1","name":"roll_dice","arguments":"{\"sides\":6}"}}`,
		`{"type":"response.completed","response":{"status":"completed"}}`,
	)
	frames := collectFrames(t, ResponsesToChatSSE(strings.NewReader(in)))

	// Reassemble tool call from frames the way the chat handler does.
	var id, name, args, finish string
	for _, f := range frames {
		choices, _ := f["choices"].([]any)
		if len(choices) == 0 {
			continue
		}
		ch := choices[0].(map[string]any)
		if fr, ok := ch["finish_reason"].(string); ok {
			finish = fr
		}
		delta, _ := ch["delta"].(map[string]any)
		tcs, _ := delta["tool_calls"].([]any)
		for _, tcAny := range tcs {
			tc := tcAny.(map[string]any)
			if v, ok := tc["id"].(string); ok && v != "" {
				id = v
			}
			fn, _ := tc["function"].(map[string]any)
			if v, ok := fn["name"].(string); ok && v != "" {
				name = v
			}
			if v, ok := fn["arguments"].(string); ok {
				args += v
			}
		}
	}
	if id != "c1" || name != "roll_dice" || args != `{"sides":6}` {
		t.Errorf("reassembled tool call wrong: id=%q name=%q args=%q", id, name, args)
	}
	if finish != "tool_calls" {
		t.Errorf("finish = %q, want tool_calls", finish)
	}
}

// reassembleToolCalls rebuilds tool calls by chat index from converted frames,
// concatenating argument fragments the way the chat handler does.
func reassembleToolCalls(t *testing.T, frames []map[string]any) map[int][3]string {
	t.Helper()
	calls := map[int][3]string{} // index → {id, name, arguments}
	for _, f := range frames {
		choices, _ := f["choices"].([]any)
		if len(choices) == 0 {
			continue
		}
		delta, _ := choices[0].(map[string]any)["delta"].(map[string]any)
		tcs, _ := delta["tool_calls"].([]any)
		for _, tcAny := range tcs {
			tc := tcAny.(map[string]any)
			k := int(tc["index"].(float64))
			c := calls[k]
			if v, ok := tc["id"].(string); ok && v != "" {
				c[0] = v
			}
			fn, _ := tc["function"].(map[string]any)
			if v, ok := fn["name"].(string); ok && v != "" {
				c[1] = v
			}
			if v, ok := fn["arguments"].(string); ok {
				c[2] += v
			}
			calls[k] = c
		}
	}
	return calls
}

// Parallel calls can arrive with their arguments only in the .done events and
// no deltas (conversation 102: every call in a batch of six ran with "").
func TestResponsesToChatSSEParallelToolCallsArgsOnlyInDone(t *testing.T) {
	in := sse(
		`{"type":"response.output_item.added","output_index":1,"item":{"type":"function_call","call_id":"a","name":"note_save","arguments":""}}`,
		`{"type":"response.output_item.added","output_index":2,"item":{"type":"function_call","call_id":"b","name":"note_list","arguments":""}}`,
		`{"type":"response.output_item.added","output_index":3,"item":{"type":"function_call","call_id":"c","name":"note_load","arguments":""}}`,
		`{"type":"response.function_call_arguments.done","output_index":1,"arguments":"{\"key\":\"c.x\",\"value\":\"v\"}"}`,
		`{"type":"response.output_item.done","output_index":1,"item":{"type":"function_call","call_id":"a","name":"note_save","arguments":"{\"key\":\"c.x\",\"value\":\"v\"}"}}`,
		// Only output_item.done carries call b's arguments.
		`{"type":"response.output_item.done","output_index":2,"item":{"type":"function_call","call_id":"b","name":"note_list","arguments":"{\"prefix\":\"c\"}"}}`,
		`{"type":"response.function_call_arguments.done","output_index":3,"arguments":"{\"key\":\"c.x\"}"}`,
		`{"type":"response.completed","response":{"status":"completed"}}`,
	)
	calls := reassembleToolCalls(t, collectFrames(t, ResponsesToChatSSE(strings.NewReader(in))))
	want := map[int][3]string{
		0: {"a", "note_save", `{"key":"c.x","value":"v"}`},
		1: {"b", "note_list", `{"prefix":"c"}`},
		2: {"c", "note_load", `{"key":"c.x"}`},
	}
	if len(calls) != len(want) {
		t.Fatalf("got %d calls, want %d: %v", len(calls), len(want), calls)
	}
	for k, w := range want {
		if calls[k] != w {
			t.Errorf("call %d = %v, want %v", k, calls[k], w)
		}
	}
}

// A call reported only by output_item.done (no .added) is emitted whole.
func TestResponsesToChatSSEToolCallWithoutAdded(t *testing.T) {
	in := sse(
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"c1","name":"get_time","arguments":"{}"}}`,
		`{"type":"response.completed","response":{"status":"completed"}}`,
	)
	frames := collectFrames(t, ResponsesToChatSSE(strings.NewReader(in)))
	calls := reassembleToolCalls(t, frames)
	if got, want := calls[0], [3]string{"c1", "get_time", "{}"}; len(calls) != 1 || got != want {
		t.Errorf("calls = %v, want only %v", calls, want)
	}
	last := frames[len(frames)-1]["choices"].([]any)[0].(map[string]any)
	if last["finish_reason"] != "tool_calls" {
		t.Errorf("finish = %v, want tool_calls", last["finish_reason"])
	}
}

// Arguments that streamed as deltas are not repeated by the .done events.
func TestResponsesToChatSSEToolArgsNotDuplicated(t *testing.T) {
	in := sse(
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"c1","name":"roll_dice"}}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"notation\":"}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"\"2d6\"}"}`,
		`{"type":"response.function_call_arguments.done","output_index":0,"arguments":"{\"notation\":\"2d6\"}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"c1","name":"roll_dice","arguments":"{\"notation\":\"2d6\"}"}}`,
		`{"type":"response.completed","response":{"status":"completed"}}`,
	)
	calls := reassembleToolCalls(t, collectFrames(t, ResponsesToChatSSE(strings.NewReader(in))))
	if got := calls[0][2]; got != `{"notation":"2d6"}` {
		t.Errorf("arguments = %q, want them once", got)
	}
}

func TestResponsesToChatSSEError(t *testing.T) {
	in := sse(`{"type":"response.failed","response":{"status":"failed"},"message":"boom"}`)
	_, err := readChatCompletionsStream(ResponsesToChatSSE(strings.NewReader(in)), nil)
	if err == nil {
		t.Fatal("expected error from failed response")
	}
}

// The Codex backend nests the reason under a top-level "error" object on a bare
// `error` frame. Regression test: this used to surface as "unknown (no message
// in error event)", hiding the real upstream failure.
func TestResponsesToChatSSEErrorFrameSurfacesMessage(t *testing.T) {
	in := sse(`{"type":"error","error":{"type":"server_error","code":"server_error","message":"An error occurred while processing your request."},"sequence_number":3}`)
	_, err := readChatCompletionsStream(ResponsesToChatSSE(strings.NewReader(in)), nil)
	if err == nil {
		t.Fatal("expected error from error frame")
	}
	if !strings.Contains(err.Error(), "An error occurred while processing your request.") {
		t.Fatalf("error message not surfaced, got: %v", err)
	}
}

// A multimodal user message (text + image_url content parts) must lower into
// Responses input_text / input_image parts, with the data URL passed through.
func TestBuildResponsesBodyMultimodalUser(t *testing.T) {
	messages := []map[string]any{
		{"role": "user", "content": []any{
			map[string]any{"type": "text", "text": "what is this?"},
			map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64,AAAA"}},
		}},
	}
	raw, err := BuildResponsesBody("gpt-x", messages, nil, nil, true)
	if err != nil {
		t.Fatalf("BuildResponsesBody: %v", err)
	}
	var req map[string]any
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}
	input := req["input"].([]any)
	if len(input) != 1 {
		t.Fatalf("expected 1 input item, got %d", len(input))
	}
	content := input[0].(map[string]any)["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 content parts, got %d: %v", len(content), content)
	}
	text := content[0].(map[string]any)
	if text["type"] != "input_text" || text["text"] != "what is this?" {
		t.Errorf("text part wrong: %v", text)
	}
	img := content[1].(map[string]any)
	if img["type"] != "input_image" || img["image_url"] != "data:image/png;base64,AAAA" {
		t.Errorf("image part wrong: %v", img)
	}
}
