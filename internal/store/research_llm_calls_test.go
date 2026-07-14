package store

import "testing"

func TestResearchLLMCallLifecycleAndAttempts(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("call-trace", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Calls", "Question", "model", "research", false, false, false, false, "", "", "", 3, 600, 0, "{}")
	if err != nil {
		t.Fatal(err)
	}

	first, err := s.BeginResearchLLMCall(job.ID, "analyzing", "synthesize", 1, "model", "https://models.example/v1", `[{"role":"user","content":"prompt"}]`, `{"max_tokens":8192}`)
	if err != nil {
		t.Fatal(err)
	}
	if first.Sequence != 1 || first.Attempt != 1 || first.Outcome != "in_progress" || first.Disposition != "pending" {
		t.Fatalf("unexpected started call: %+v", first)
	}
	price := 0.25
	if err := s.CompleteResearchLLMCall(first.ID, 1234, "partial response", "length", `{"completion_tokens":8192}`, &price, 200, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.SetResearchLLMCallDisposition(first.ID, "rejected"); err != nil {
		t.Fatal(err)
	}
	second, err := s.BeginResearchLLMCall(job.ID, "analyzing", "synthesize", 1, "model", "https://models.example/v1", `[]`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if second.Sequence != 2 || second.Attempt != 2 {
		t.Fatalf("retry numbering = sequence %d attempt %d", second.Sequence, second.Attempt)
	}

	calls, err := s.ListResearchLLMCalls(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("call count = %d", len(calls))
	}
	completed := calls[0]
	if completed.Outcome != "succeeded" || completed.Disposition != "rejected" || completed.FinishReason == nil || *completed.FinishReason != "length" {
		t.Fatalf("completed call metadata = %+v", completed)
	}
	if completed.Response == nil || *completed.Response != "partial response" || completed.Usage == nil || completed.PriceUSD == nil {
		t.Fatalf("completed call output missing: %+v", completed)
	}
}
