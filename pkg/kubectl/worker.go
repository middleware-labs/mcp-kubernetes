package kubectl

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/Azure/mcp-kubernetes/pkg/kubectl/ws"
)

var (
	errInvalidMode = errors.New("invalid mode passed")
)

type ClusterRoleCheckResult struct {
	Success          bool   `json:"success"`
	HasAdminRole     bool   `json:"has_admin_role"`
	ErrorType        string `json:"error_type,omitempty"`
	ErrorMessage     string `json:"error_message,omitempty"`
	ClusterRoleFound bool   `json:"cluster_role_found"`
	ResponseReceived bool   `json:"response_received"`
}

type Mode uint16

var (
	ModeLocation Mode = 0
	ModeAgent    Mode = 1
)

func (m Mode) String() string {
	switch m {
	case ModeLocation:
		return "location"
	case ModeAgent:
		return "agent"
	}
	return "unknown"
}

type Config struct {
	Mode                Mode
	Location            string
	AccountUID          string
	Hostname            string
	PulsarHost          string
	UnsubscribeEndpoint string
	NCAPassword         string
	Token               string
	Timeout             int
	CaptureEndpoint     string
	Fingerprint         string
}

type Worker struct {
	cfg          *Config
	pulsarClient *ws.Client
	topic        string
	consumer     ws.Consumer
	messages     map[string]*ws.Msg
	messagesLock sync.Mutex
	pending      sync.Map
}

func New(cfg *Config) (*Worker, error) {
	var topic string
	switch cfg.Mode {
	case ModeLocation:
		topic = fmt.Sprintf("%s-%s", ModeLocation, strings.ToLower(cfg.Location))
	case ModeAgent:
		if cfg.Location == "" {
			topic = fmt.Sprintf("%s-%s", ModeAgent, strings.ToLower(cfg.Token))
		} else {
			topic = fmt.Sprintf("%s-%s-%x", ModeAgent, strings.ToLower(cfg.Token),
				sha1.Sum([]byte(strings.ToLower(cfg.Location))))
		}
	default:
		return &Worker{}, errInvalidMode
	}

	// LOG: dump full config (mask token) and computed topic at startup
	slog.Info("worker created",
		"mode", cfg.Mode.String(),
		"location", cfg.Location,
		"account_uid", cfg.AccountUID,
		"hostname", cfg.Hostname,
		"pulsar_host", cfg.PulsarHost,
		"unsubscribe_endpoint", cfg.UnsubscribeEndpoint,
		"token_prefix", safePrefix(cfg.Token, 8),
		"timeout", cfg.Timeout,
		"fingerprint", cfg.Fingerprint,
		"computed_topic", topic)

	return &Worker{
		cfg:          cfg,
		pulsarClient: ws.New(cfg.PulsarHost),
		topic:        topic,
		messages:     make(map[string]*ws.Msg),
	}, nil
}

// safePrefix returns the first n chars of s for logging without leaking secrets
func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (w *Worker) GetMessage(key string) (*ws.Msg, bool) {
	w.messagesLock.Lock()
	defer w.messagesLock.Unlock()
	msg, ok := w.messages[key]
	return msg, ok
}

func (w *Worker) DeleteMessage(key string) {
	w.messagesLock.Lock()
	defer w.messagesLock.Unlock()
	delete(w.messages, key)
}

func (w *Worker) SetMessage(key string, msg *ws.Msg) {
	w.messagesLock.Lock()
	defer w.messagesLock.Unlock()
	w.messages[key] = msg
}

func (w *Worker) StartSubscriber(topic string) error {
	// LOG: who is calling and with what topic
	slog.Info("StartSubscriber called",
		"requested_topic", topic,
		"worker_topic", w.topic,
		"topics_match", topic == w.topic,
		"fingerprint", w.cfg.Fingerprint)
	return w.startSubscriberWithRetry(topic, 0)
}

func (w *Worker) startSubscriberWithRetry(topic string, attempt int) error {
	fullTopic := "persistent/public/default/" + topic
	subName := "subscribe-" + w.cfg.Fingerprint

	// LOG: exact pulsar topic + subscription name being used
	slog.Info("creating pulsar consumer",
		"full_topic", fullTopic,
		"subscription_name", subName,
		"subscription_type", "Shared",
		"attempt", attempt)

	consumer, err := w.pulsarClient.Consumer(
		fullTopic,
		subName,
		ws.Params{
			"subscriptionType": "Shared",
			"token":            w.cfg.Token,
		})
	if err != nil {
		backoff := time.Second * time.Duration(1<<attempt)
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		slog.Error("failed to create consumer, retrying...",
			"err", err, "attempt", attempt, "backoff", backoff, "topic", fullTopic)
		time.Sleep(backoff)
		return w.startSubscriberWithRetry(topic, attempt+1)
	}

	w.consumer = consumer
	slog.Info("subscriber started successfully", "topic", topic, "full_topic", fullTopic)

	go func() {
		for {
			ctx := context.Background()

			// LOG: heartbeat that we are actively waiting
			slog.Debug("subscriber waiting for next message", "topic", topic)

			msg, err := consumer.Receive(ctx)
			if err != nil {
				slog.Error("consumer receive error",
					"err", err, "topic", topic, "full_topic", fullTopic)
				consumer.Close()
				_ = w.startSubscriberWithRetry(topic, 0)
				return
			}

			// LOG: every raw message received, before parsing
			slog.Info("RAW message received from pulsar",
				"topic", topic,
				"payload_size", len(msg.Payload),
				"payload_preview", safePrefix(string(msg.Payload), 500))

			var payload struct {
				Id     int                    `json:"Id"`
				Result map[string]interface{} `json:"result"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				// LOG: malformed payload, dump the whole thing
				slog.Error("failed to unmarshal payload",
					"err", err,
					"raw_payload", string(msg.Payload))
				w.retryAck(ctx, consumer, msg)
				continue
			}

			// LOG: parsed ID + a snapshot of all pending IDs we are waiting on
			pendingIDs := w.snapshotPendingIDs()
			slog.Info("parsed message",
				"received_id", payload.Id,
				"received_id_type", fmt.Sprintf("%T", payload.Id),
				"result_keys", mapKeys(payload.Result),
				"pending_ids", pendingIDs,
				"id_matches_pending", contains(pendingIDs, payload.Id))

			if chAny, ok := w.pending.Load(payload.Id); ok {
				if ch, ok := chAny.(chan string); ok {
					if stdout, ok := payload.Result["stdout"].(string); ok {
						slog.Info("delivering response to waiter",
							"id", payload.Id,
							"stdout_size", len(stdout),
							"stdout_preview", safePrefix(stdout, 200))
						ch <- stdout
					} else {
						// LOG: stdout missing or wrong type
						slog.Warn("result.stdout missing or not string, delivering empty",
							"id", payload.Id,
							"result", payload.Result)
						ch <- ""
					}
					close(ch)
				} else {
					slog.Error("pending entry is not chan string",
						"id", payload.Id,
						"actual_type", fmt.Sprintf("%T", chAny))
				}
				w.pending.Delete(payload.Id)
				w.retryAck(ctx, consumer, msg)
				continue
			}

			// LOG: message arrived but nobody is waiting for this ID. Major clue.
			slog.Warn("orphan message (no pending waiter), nacking",
				"received_id", payload.Id,
				"pending_ids", pendingIDs)
			w.retryNack(ctx, consumer, msg)
		}
	}()

	return nil
}

// snapshotPendingIDs walks the sync.Map and returns currently-waiting IDs
func (w *Worker) snapshotPendingIDs() []int {
	var ids []int
	w.pending.Range(func(k, _ interface{}) bool {
		if id, ok := k.(int); ok {
			ids = append(ids, id)
		}
		return true
	})
	return ids
}

func contains(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (w *Worker) retryAck(ctx context.Context, consumer ws.Consumer, msg *ws.Msg) {
	for {
		if err := consumer.Ack(ctx, msg); err != nil {
			slog.Error("ack failed, retrying in 1s", "err", err)
			time.Sleep(1 * time.Second)
			continue
		}
		return
	}
}

func (w *Worker) retryNack(ctx context.Context, consumer ws.Consumer, msg *ws.Msg) {
	for {
		if err := consumer.Nack(ctx, msg); err != nil {
			slog.Error("nack failed, retrying in 1s", "err", err)
			time.Sleep(1 * time.Second)
			continue
		}
		return
	}
}

func (w *Worker) produceMessage(accountUid string,
	topic string, key string, payload map[string]interface{}) error {

	type topicKeyPayload struct {
		Topic   string
		Key     string
		Payload map[string]interface{}
	}

	pay := topicKeyPayload{
		Topic:   topic,
		Key:     key,
		Payload: payload,
	}
	str, _ := json.Marshal(pay)
	url := strings.ReplaceAll(w.cfg.UnsubscribeEndpoint, "{ACC}", accountUid)

	// LOG: exact URL, topic, key, and payload going out
	slog.Info("PRODUCE -> publishing message",
		"url", url,
		"topic", topic,
		"key", key,
		"payload_size", len(str),
		"payload", string(str))

	req, err := http.NewRequest("POST", url, bytes.NewReader(str))
	if err != nil {
		slog.Error("failed to create request", "error", err.Error())
		return fmt.Errorf("failed to create request: %s", err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	if w.cfg.NCAPassword != "" {
		req.Header.Set("Authorization", w.cfg.NCAPassword)
	}

	start := time.Now()
	re, err := http.DefaultClient.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		slog.Error("PRODUCE failed (transport)",
			"error", err.Error(), "url", url, "elapsed_ms", elapsed.Milliseconds())
		return fmt.Errorf("failed to produce message: %s", err.Error())
	}
	defer re.Body.Close()

	bodyBytes, _ := io.ReadAll(re.Body)

	if re.StatusCode != 200 {
		slog.Error("PRODUCE failed (status)",
			"status", re.Status,
			"url", url,
			"response_body", string(bodyBytes),
			"elapsed_ms", elapsed.Milliseconds())
		return fmt.Errorf("failed to produce message: %s", string(bodyBytes))
	}

	// LOG: success with full response body so we can confirm broker accepted it
	slog.Info("PRODUCE -> success",
		"status", re.Status,
		"elapsed_ms", elapsed.Milliseconds(),
		"response_body", string(bodyBytes))
	return nil
}

func (w *Worker) sendRequest(accountUid string, id int, topic string, payload map[string]interface{}) error {
	idString := fmt.Sprintf("%d", id)
	payloadMap := map[string]interface{}{
		"Not":         "",
		"Action":      "mcp-k8s",
		"Id":          id,
		"AccountUID":  accountUid,
		"preview_id":  idString,
		"account_uid": accountUid,
		"topic":       topic,
		"result":      payload,
	}

	// LOG: confirm what topic the request is going to vs what the worker subscribed to
	slog.Info("sendRequest preparing",
		"id", id,
		"send_topic", topic,
		"worker_topic", w.topic,
		"topics_match", topic == w.topic,
		"account_uid", accountUid)

	return w.produceMessage(accountUid, topic, idString, payloadMap)
}

func (w *Worker) CheckClusterRolePermission(timeout int) *ClusterRoleCheckResult {
	cmd := "kubectl get clusterroles"

	id := int(time.Now().UnixMilli())
	respCh := make(chan string, 1)
	w.pending.Store(id, respCh)

	topic := fmt.Sprintf("mcp-%s-%x",
		strings.ToLower(w.cfg.Token),
		sha1.Sum([]byte(strings.ToLower(w.cfg.Location))))

	// LOG: cluster-role-check entry point
	slog.Info("CheckClusterRolePermission start",
		"id", id,
		"topic", topic,
		"worker_topic", w.topic,
		"topics_match", topic == w.topic,
		"timeout", timeout)

	err := w.sendRequest(w.cfg.AccountUID, id, topic, map[string]interface{}{
		"command": cmd,
	})
	if err != nil {
		w.pending.Delete(id)
		slog.Error("failed to send cluster role check request", "error", err, "id", id, "topic", topic)
		return &ClusterRoleCheckResult{
			Success:          false,
			HasAdminRole:     false,
			ErrorType:        "connection",
			ErrorMessage:     fmt.Sprintf("failed to send cluster role check: %s", err.Error()),
			ClusterRoleFound: false,
			ResponseReceived: false,
		}
	}

	slog.Info("CheckClusterRolePermission waiting", "id", id, "topic", topic, "timeout_seconds", timeout)

	var res string
	select {
	case res = <-respCh:
		slog.Info("CheckClusterRolePermission received response",
			"id", id,
			"response_size", len(res),
			"response_preview", safePrefix(res, 300))
		if strings.Contains(res, "mw-opsai-cluster-role") {
			slog.Info("mw-opsai-cluster-role FOUND")
			return &ClusterRoleCheckResult{
				Success: true, HasAdminRole: true,
				ClusterRoleFound: true, ResponseReceived: true,
			}
		}
		slog.Info("mw-opsai-cluster-role NOT FOUND in response")
		return &ClusterRoleCheckResult{
			Success: true, HasAdminRole: false,
			ClusterRoleFound: false, ResponseReceived: true,
		}

	case <-time.After(time.Second * time.Duration(timeout)):
		w.pending.Delete(id)
		// LOG: timeout with full diagnostic context
		slog.Error("CheckClusterRolePermission TIMEOUT",
			"id", id,
			"topic", topic,
			"worker_topic", w.topic,
			"timeout_seconds", timeout,
			"still_pending", w.snapshotPendingIDs())
		return &ClusterRoleCheckResult{
			Success: false, HasAdminRole: false,
			ErrorType:    "timeout",
			ErrorMessage: fmt.Sprintf("timeout checking cluster roles after %d seconds", timeout),
		}
	}
}
