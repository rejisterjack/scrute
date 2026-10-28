package anthropic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompleteParsesToolUse(t *testing.T) {
	const key = "sk-ant-test-secret-value"
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		if r.Header.Get("x-api-key") != key {
			t.Errorf("missing api key header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
		  "id": "msg_test",
		  "type": "message",
		  "role": "assistant",
		  "model": "claude-sonnet-5",
		  "stop_reason": "tool_use",
		  "content": [{
		    "type": "tool_use",
		    "id": "toolu_test",
		    "name": "submit_review",
		    "input": {"summary":"Found a bug.","issues":[{"severity":"critical","file":"a.go","line":3,"title":"nil deref","detail":"x is nil"}]}
		  }],
		  "usage": {"input_tokens": 11, "output_tokens": 7, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
		}`)
	}))
	defer server.Close()

	raw, inTok, outTok, err := New(key, WithBaseURL(server.URL), WithMaxRetries(0)).Complete(
		context.Background(), "claude-sonnet-5", "system", "review a.go please",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "nil deref") {
		t.Fatalf("raw %s", raw)
	}
	if inTok != 11 || outTok != 7 {
		t.Fatalf("tokens %d %d", inTok, outTok)
	}
	if !strings.Contains(body, "submit_review") || !strings.Contains(body, "review a.go please") || !strings.Contains(body, "claude-sonnet-5") {
		t.Fatalf("request body missing review contract:\n%s", body)
	}
	if strings.Contains(body, key) {
		t.Fatal("API key was copied into the request body")
	}
}

func TestCompleteRedactsKey(t *testing.T) {
	const key = "sk-ant-test-secret-value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"rejected `+r.Header.Get("x-api-key")+`"}}`)
	}))
	defer server.Close()

	_, _, _, err := New(key, WithBaseURL(server.URL), WithMaxRetries(0)).Complete(
		context.Background(), "claude-sonnet-5", "system", "user",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), key) {
		t.Fatalf("key leaked: %s", err)
	}
}
