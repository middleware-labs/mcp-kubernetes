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
	"sync/atomic"
	"time"

	"log/slog"

	"github.com/Azure/mcp-kubernetes/pkg/kubectl/ws"
)

var (
	errInvalidMode   = errors.New("invalid mode passed")
	errEmptyLocation = errors.New("location is empty: the request topic is derived from it, " +
		"so an empty value would collide with every other deployment sharing this token")
)

// requestIDCounter backs nextRequestID. It is seeded from the wall clock so ids
// stay roughly ordered across restarts.
var requestIDCounter atomic.Int64

func init() {
	requestIDCounter.Store(time.Now().UnixMilli())
}

// nextRequestID returns a process-unique correlation id for a remote command.
//
// This used to be int(time.Now().UnixMilli()), which meant two calls issued in
// the same millisecond produced the same id. The second one overwrote the first
// in Worker.pending, so the first caller's channel was never signalled and it
// blocked until its timeout even though a valid response had come back.
func nextRequestID() int {
	return int(requestIDCounter.Add(1))
}

// MCPTopic returns the request topic used to reach a synthetics-agent running
// in MCP mode. The agent subscribes to this exact name and publishes replies to
// MCPTopic()+"-unsubscribe".
//
// Both sides hash a cluster name into the topic and there is no handshake, so a
// mismatch is silent: the publish succeeds, nothing consumes it, and the caller
// simply times out. Keep this the single definition of the name — do not inline
// the format string at call sites.
func MCPTopic(token, location string) string {
	return fmt.Sprintf("mcp-%s-%x", strings.ToLower(token),
		sha1.Sum([]byte(strings.ToLower(location))))
}

// Topic returns this worker's request topic.
func (w *Worker) Topic() string {
	return MCPTopic(w.cfg.Token, w.cfg.Location)
}

// ReplyTopic returns the topic the agent publishes command results back on.
func (w *Worker) ReplyTopic() string {
	return w.Topic() + "-unsubscribe"
}

type ClusterRoleCheckResult struct {
	Success          bool   `json:"success"`
	HasAdminRole     bool   `json:"has_admin_role"`
	ErrorType        string `json:"error_type,omitempty"` // "connection", "timeout", "permission", "other"
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

// Worker is the main worker struct
type Worker struct {
	cfg          *Config
	pulsarClient *ws.Client
	topic        string
	consumer     ws.Consumer
	messages     map[string]*ws.Msg
	messagesLock sync.Mutex
	pending      sync.Map
}

// New creates a new worker
func New(cfg *Config) (*Worker, error) {
	// Every remote command is routed on a topic derived from cfg.Location. An
	// empty value hashes to the sha1 of "", a constant, so the worker would
	// quietly share a topic with every other deployment using the same token.
	if strings.TrimSpace(cfg.Location) == "" {
		return &Worker{}, errEmptyLocation
	}

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

	return &Worker{
		cfg:          cfg,
		pulsarClient: ws.New(cfg.PulsarHost),
		topic:        topic,
		messages:     make(map[string]*ws.Msg),
	}, nil
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
	return w.startSubscriberWithRetry(topic, 0)
}

func (w *Worker) startSubscriberWithRetry(topic string, attempt int) error {
	consumer, err := w.pulsarClient.Consumer(
		"persistent/public/default/"+topic,
		"subscribe-"+w.cfg.Fingerprint,
		ws.Params{
			"subscriptionType": "Shared",
			"token":            w.cfg.Token,
		})
	if err != nil {
		backoff := time.Second * time.Duration(1<<attempt) // exponential backoff
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		slog.Error("failed to create consumer, retrying...",
			"err", err, "attempt", attempt, "backoff", backoff)

		time.Sleep(backoff)
		return w.startSubscriberWithRetry(topic, attempt+1)
	}

	w.consumer = consumer
	slog.Info("started subscriber", "topic", redactTopic(topic))

	go func() {
		for {
			ctx := context.Background()
			msg, err := consumer.Receive(ctx)
			if err != nil {
				slog.Error("consumer receive error", "err", err)
				consumer.Close()

				// reconnect (restart fresh)
				_ = w.startSubscriberWithRetry(topic, 0)
				return
			}

			var payload struct {
				Id     int                    `json:"Id"`
				Result map[string]interface{} `json:"result"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				w.retryAck(ctx, consumer, msg)
				continue
			}

			if chAny, ok := w.pending.Load(payload.Id); ok {
				if ch, ok := chAny.(chan string); ok {
					if stdout, ok := payload.Result["stdout"].(string); ok {
						slog.Info("received response", slog.Int("id", payload.Id))
						ch <- stdout
					} else {
						ch <- ""
					}
					close(ch)
				}
				w.pending.Delete(payload.Id)
				w.retryAck(ctx, consumer, msg)
				continue
			}

			w.retryAck(ctx, consumer, msg)
		}
	}()

	return nil
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

// redactTopic strips the API key from a topic name so that it is safe to log.
func redactTopic(topic string) string {
	parts := strings.SplitN(topic, "-", 3)
	if len(parts) < 2 {
		return topic
	}
	parts[1] = "<redacted>"
	return strings.Join(parts, "-")
}

// produceMessage publishes and returns how many consumers the broker reported
// for the topic, or nil when it reported none (older bifrost or bridge).
func (w *Worker) produceMessage(accountUid string,
	topic string, key string, payload map[string]interface{}, expirationMs string) (*int, error) {

	type topicKeyPayload struct {
		Topic      string
		Key        string
		Payload    map[string]interface{}
		Expiration string `json:",omitempty"`
	}

	pay := topicKeyPayload{
		Topic:      topic,
		Key:        key,
		Payload:    payload,
		Expiration: expirationMs,
	}
	str, _ := json.Marshal(pay)
	slog.Info("producing message", slog.String("url", w.cfg.UnsubscribeEndpoint),
		slog.String("topic", redactTopic(topic)), slog.String("key", key))
	url := strings.ReplaceAll(w.cfg.UnsubscribeEndpoint, "{ACC}", accountUid)
	req, err := http.NewRequest("POST", url, bytes.NewReader(str))
	if err != nil {
		slog.Error("failed to create request", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to create request: %s", err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	if w.cfg.NCAPassword != "" {
		req.Header.Set("Authorization", w.cfg.NCAPassword)
	}

	re, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to produce message", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to produce message: %s", err.Error())
	}
	defer re.Body.Close()

	if re.StatusCode != 200 {
		str, _ := io.ReadAll(re.Body)
		slog.Error("failed to produce message", slog.String("response status", re.Status),
			slog.String("url", url), slog.String("response", string(str)))
		return nil, fmt.Errorf("failed to produce message: %s", string(str))
	}

	var published struct {
		Consumers *int `json:"consumers"`
	}
	// The message is already published, so an unreadable body only means the
	// consumer count is unknown.
	_ = json.NewDecoder(re.Body).Decode(&published)
	return published.Consumers, nil
}

// sendRequest publishes a command for the agent. noSubscriber is true only when
// the broker positively reported that nobody is consuming the request topic.
func (w *Worker) sendRequest(accountUid string, id int, topic string, payload map[string]interface{}) (noSubscriber bool, err error) {
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
	ttlMs := fmt.Sprintf("%d", (w.cfg.Timeout)*1000)
	consumers, err := w.produceMessage(accountUid, topic, idString, payloadMap, ttlMs)
	return consumers != nil && *consumers == 0, err
}

// noSubscriberGrace is how long to wait for a reply once the broker has said
// nobody is subscribed. It only covers an agent that is mid-reconnect, which
// retries every 5s, so a missing agent is reported well before the full timeout.
var noSubscriberGrace = 10 * time.Second

// replyWait returns how long to wait for the reply to a request.
func replyWait(timeoutSeconds int, noSubscriber bool) time.Duration {
	wait := time.Second * time.Duration(timeoutSeconds)
	if noSubscriber && wait > noSubscriberGrace {
		return noSubscriberGrace
	}
	return wait
}

// noSubscriberError reports a request that nothing could have received. It is
// worded for the investigating agent, which otherwise retries a dead cluster.
func (w *Worker) noSubscriberError() error {
	return fmt.Errorf("no agent is connected for cluster %q: nothing is subscribed to its request topic. "+
		"Retrying will not help; check that the opsai deployment in the cluster is running and can reach the broker",
		w.cfg.Location)
}

// CheckClusterRolePermission validates if mw-opsai-cluster-role exists
func (w *Worker) CheckClusterRolePermission(timeout int) *ClusterRoleCheckResult {
	cmd := "kubectl get clusterroles"

	id := nextRequestID()
	respCh := make(chan string, 1)
	w.pending.Store(id, respCh)

	topic := w.Topic()

	noSubscriber, err := w.sendRequest(w.cfg.AccountUID, id, topic, map[string]interface{}{
		"command": cmd,
	})
	if err != nil {
		slog.Error("failed to send cluster role check request", "error", err, "id", id, "topic", redactTopic(topic))
		return &ClusterRoleCheckResult{
			Success:          false,
			HasAdminRole:     false,
			ErrorType:        "connection",
			ErrorMessage:     fmt.Sprintf("failed to send cluster role check: %s", err.Error()),
			ClusterRoleFound: false,
			ResponseReceived: false,
		}
	}

	slog.Info("checking for mw-opsai-cluster-role", "id", id, "topic", redactTopic(topic))

	var res string
	select {
	case res = <-respCh:
		slog.Info("received cluster roles response", "id", id)
		if strings.Contains(res, "mw-opsai-cluster-role") {
			slog.Info("mw-opsai-cluster-role found - admin/write permission available")
			return &ClusterRoleCheckResult{
				Success:          true,
				HasAdminRole:     true,
				ErrorType:        "",
				ErrorMessage:     "",
				ClusterRoleFound: true,
				ResponseReceived: true,
			}
		}
		slog.Info("mw-opsai-cluster-role not found - using readonly permission")
		return &ClusterRoleCheckResult{
			Success:          true,
			HasAdminRole:     false,
			ErrorType:        "",
			ErrorMessage:     "",
			ClusterRoleFound: false,
			ResponseReceived: true,
		}

	case <-time.After(replyWait(timeout, noSubscriber)):
		w.pending.Delete(id)
		slog.Error("timeout checking cluster roles", "id", id, "topic", redactTopic(topic))
		if noSubscriber {
			return &ClusterRoleCheckResult{
				Success:          false,
				HasAdminRole:     false,
				ErrorType:        "connection",
				ErrorMessage:     w.noSubscriberError().Error(),
				ClusterRoleFound: false,
				ResponseReceived: false,
			}
		}
		return &ClusterRoleCheckResult{
			Success:          false,
			HasAdminRole:     false,
			ErrorType:        "timeout",
			ErrorMessage:     fmt.Sprintf("timeout checking cluster roles after %d seconds", timeout),
			ClusterRoleFound: false,
			ResponseReceived: false,
		}
	}
}
