package kubectl

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testToken = "SecretApiKey"

// publishCapture is what the fake bifrost publish endpoint received.
type publishCapture struct {
	Topic      string
	Key        string
	Expiration string
}

func newRequestWorker(t *testing.T, timeout int, response string, onPublish func(w *Worker, got publishCapture)) *Worker {
	t.Helper()
	worker := &Worker{cfg: &Config{
		Token:      testToken,
		Location:   "my-k8s-cluster",
		AccountUID: "acc",
		Timeout:    timeout,
	}}
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var got publishCapture
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode publish body: %v", err)
		}
		if onPublish != nil {
			onPublish(worker, got)
		}
		_, _ = rw.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)
	worker.cfg.UnsubscribeEndpoint = srv.URL
	return worker
}

func shortenGrace(t *testing.T, grace time.Duration) {
	t.Helper()
	previous := noSubscriberGrace
	noSubscriberGrace = grace
	t.Cleanup(func() { noSubscriberGrace = previous })
}

func TestRedactTopic(t *testing.T) {
	tests := map[string]string{
		"mcp-" + testToken + "-abc123":             "mcp-<redacted>-abc123",
		"mcp-" + testToken + "-abc123-unsubscribe": "mcp-<redacted>-abc123-unsubscribe",
		"agent-" + testToken:                       "agent-<redacted>",
		"preview":                                  "preview",
	}
	for topic, want := range tests {
		got := redactTopic(topic)
		if got != want {
			t.Errorf("redactTopic(%q) = %q, want %q", topic, got, want)
		}
		if strings.Contains(got, testToken) {
			t.Errorf("redactTopic(%q) leaked the key: %q", topic, got)
		}
	}
}

func TestReplyWait(t *testing.T) {
	shortenGrace(t, 10*time.Second)
	tests := []struct {
		name         string
		timeout      int
		noSubscriber bool
		want         time.Duration
	}{
		{"subscriber present keeps the full timeout", 60, false, 60 * time.Second},
		{"no subscriber waits only the grace", 60, true, 10 * time.Second},
		{"grace never extends a shorter timeout", 3, true, 3 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := replyWait(tt.timeout, tt.noSubscriber); got != tt.want {
				t.Errorf("replyWait = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendRequestReadsConsumers(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
	}{
		{"zero consumers", `{"status":true,"message":"done","consumers":0}`, true},
		{"one consumer", `{"status":true,"message":"done","consumers":1}`, false},
		{"older bifrost reports nothing", `{"status":true,"message":"done"}`, false},
		{"unreadable body is unknown, not missing", `not json`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got publishCapture
			worker := newRequestWorker(t, 60, tt.response, func(_ *Worker, published publishCapture) { got = published })

			noSubscriber, err := worker.sendRequest("acc", 7, worker.Topic(), map[string]interface{}{"command": "kubectl get nodes"})
			if err != nil {
				t.Fatalf("sendRequest: %v", err)
			}
			if noSubscriber != tt.want {
				t.Errorf("noSubscriber = %v, want %v", noSubscriber, tt.want)
			}
			if got.Expiration != "60000" {
				t.Errorf("Expiration = %q, want the caller's 60s timeout", got.Expiration)
			}
		})
	}
}

func TestExecuteFailsFastWithoutSubscriber(t *testing.T) {
	shortenGrace(t, 50*time.Millisecond)
	worker := newRequestWorker(t, 30, `{"status":true,"consumers":0}`, nil)
	executor := NewExecutor(worker)

	start := time.Now()
	_, err := executor.executeKubectlCommandOnHost("get nodes", "", nil)
	if err == nil {
		t.Fatal("expected an error when nothing is subscribed")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("took %v, should return after the grace rather than the 30s timeout", elapsed)
	}
	if !strings.Contains(err.Error(), `no agent is connected for cluster "my-k8s-cluster"`) {
		t.Errorf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), testToken) {
		t.Errorf("error leaked the key: %v", err)
	}
}

func TestExecuteAcceptsReplyDuringGrace(t *testing.T) {
	shortenGrace(t, 5*time.Second)
	// An agent mid-reconnect: the broker saw no consumer at publish time, but the
	// agent picked the message up moments later and replied.
	worker := newRequestWorker(t, 30, `{"status":true,"consumers":0}`, func(w *Worker, got publishCapture) {
		id, err := strconv.Atoi(got.Key)
		if err != nil {
			t.Errorf("publish key %q: %v", got.Key, err)
			return
		}
		go func() {
			time.Sleep(20 * time.Millisecond)
			if ch, ok := w.pending.Load(id); ok {
				ch.(chan string) <- "node-1 Ready"
			}
		}()
	})

	got, err := NewExecutor(worker).executeKubectlCommandOnHost("get nodes", "", nil)
	if err != nil {
		t.Fatalf("expected the late reply to win: %v", err)
	}
	if got != "node-1 Ready" {
		t.Errorf("got %q", got)
	}
}

func TestExecuteKeepsFullTimeoutWhenConsumersUnknown(t *testing.T) {
	shortenGrace(t, 10*time.Millisecond)
	worker := newRequestWorker(t, 1, `{"status":true,"message":"done"}`, nil)

	start := time.Now()
	_, err := NewExecutor(worker).executeKubectlCommandOnHost("get nodes", "", nil)
	if err == nil || !strings.Contains(err.Error(), "timeout waiting for response after 1s") {
		t.Fatalf("expected the existing timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Errorf("returned after %v, must not apply the grace when the broker reported nothing", elapsed)
	}
}

func TestClusterRoleCheckReportsNoSubscriberAsConnectionIssue(t *testing.T) {
	shortenGrace(t, 20*time.Millisecond)
	worker := newRequestWorker(t, 30, `{"status":true,"consumers":0}`, nil)

	result := worker.CheckClusterRolePermission(30)
	if result.Success {
		t.Fatal("expected a failed check")
	}
	if result.ErrorType != "connection" {
		t.Errorf("ErrorType = %q, want connection so the access level is kept", result.ErrorType)
	}
	if !strings.Contains(result.ErrorMessage, "no agent is connected") {
		t.Errorf("unexpected message: %q", result.ErrorMessage)
	}
}
