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
	slog.Info("started subscriber", "topic", topic)

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

			w.retryNack(ctx, consumer, msg)
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
	slog.Info("producing message", slog.String("url", w.cfg.UnsubscribeEndpoint), slog.String("payload", string(str)))
	url := strings.ReplaceAll(w.cfg.UnsubscribeEndpoint, "{ACC}", accountUid)
	req, err := http.NewRequest("POST", url, bytes.NewReader(str))
	if err != nil {
		slog.Error("failed to create request", slog.String("error", err.Error()))
		return fmt.Errorf("failed to create request: %s", err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	if w.cfg.NCAPassword != "" {
		req.Header.Set("Authorization", w.cfg.NCAPassword)
	}

	re, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to produce message", slog.String("error", err.Error()))
		return fmt.Errorf("failed to produce message: %s", err.Error())
	}

	if re.StatusCode != 200 {
		str, _ := io.ReadAll(re.Body)
		slog.Error("failed to produce message", slog.String("response status", re.Status),
			slog.String("url", url), slog.String("response", string(str)))
		return fmt.Errorf("failed to produce message: %s", string(str))
	}
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
	return w.produceMessage(accountUid, topic, idString, payloadMap)
}

// CheckClusterRolePermission validates if mw-opsai-cluster-role exists
func (w *Worker) CheckClusterRolePermission(timeout int) *ClusterRoleCheckResult {
	cmd := "kubectl get clusterroles"

	id := int(time.Now().UnixMilli())
	respCh := make(chan string, 1)
	w.pending.Store(id, respCh)

	topic := fmt.Sprintf("mcp-%s-%x",
		strings.ToLower(w.cfg.Token),
		sha1.Sum([]byte(strings.ToLower(w.cfg.Location))))

	err := w.sendRequest(w.cfg.AccountUID, id, topic, map[string]interface{}{
		"command": cmd,
	})
	if err != nil {
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

	slog.Info("checking for mw-opsai-cluster-role", "id", id, "topic", topic)

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

	case <-time.After(time.Second * time.Duration(timeout)):
		w.pending.Delete(id)
		slog.Error("timeout checking cluster roles", "id", id, "topic", topic)
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
