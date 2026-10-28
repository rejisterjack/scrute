package openai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompleteParsesJSONSchema(t *testing.T) {
	const key = "sk-openai-test-secret-value"
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		if r.Header.Get("Authorization") != "Bearer "+key {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
		  "id": "chatcmpl-test",
		  "object": "chat.completion",
		  "created": 1,
		  "model": "gpt-5.4",
		  "choices": [{
		    "index": 0,
		    "finish_reason": "stop",
		    "message": {"role": "assistant", "content": "{\"summary\":\"Looks fine.\",\"issues\":[]}"}
		  }],
		  "usage": {"prompt_tokens": 5, "completion_tokens": 6, "total_tokens": 11}
		}`)
	}))
	defer server.Close()

	raw, inTok, outTok, err := New(key, WithBaseURL(server.URL), WithMaxRetries(0)).Complete(
		context.Background(), "gpt-5.4", "system", "review main.go",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "Looks fine.") {
		t.Fatalf("raw %s", raw)
	}
	if inTok != 5 || outTok != 6 {
		t.Fatalf("tokens %d %d", inTok, outTok)
	}
	if !strings.Contains(body, "scrute_review") || !strings.Contains(body, "json_schema") || !strings.Contains(body, "review main.go") {
		t.Fatalf("request body missing schema contract:\n%s", body)
	}
	if strings.Contains(body, key) {
		t.Fatal("API key was copied into the request body")
	}
}

func TestCompleteRedactsKey(t *testing.T) {
	const key = "sk-openai-test-secret-value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"message":"bad `+key+`"}}`)
	}))
	defer server.Close()

	_, _, _, err := New(key, WithBaseURL(server.URL), WithMaxRetries(0)).Complete(
		context.Background(), "gpt-5.4", "system", "user",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), key) {
		t.Fatalf("key leaked: %s", err)
	}
}
