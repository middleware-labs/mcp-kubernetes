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
	"strconv"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/Azure/mcp-kubernetes/pkg/kubectl/ws"
)

var (
	errInvalidMode = errors.New("invalid mode passed")
)

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
	CaptureEndpoint     string
}

// Worker is the main worker struct
type Worker struct {
	cfg          *Config
	pulsarClient *ws.Client
	topic        string
	consumer     ws.Consumer
	messages     map[string]*ws.Msg
	messagesLock sync.Mutex
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
func (w *Worker) SubscribeUpdates(topic string, token string, id int) (string, error) {
	// instanceId := strings.ToLower(os.Getenv("HOSTNAME"))

	consumerName := "subscribe-" + w.cfg.Hostname + "-" + strconv.FormatInt(time.Now().UTC().Unix(), 10)
	url := w.cfg.Hostname + "/consumer/persistent/public/default/" +
		topic + "/" + consumerName + "?token=" + token
	slog.Info("subscribing to topic", slog.String("url", url),
		slog.String("consumer", consumerName), slog.String("token", token))

	var err error
	w.consumer, err = w.pulsarClient.Consumer("persistent/public/default/"+topic,
		"subscribe", ws.Params{
			"subscriptionType":           "Shared",
			"ackTimeoutMillis":           strconv.Itoa(60 * 60 * 1000),
			"consumerName":               consumerName,
			"negativeAckRedeliveryDelay": "0",
			"pullMode":                   "false",
			"receiverQueueSize":          "500000",
			"token":                      token,
		})
	if err != nil {
		slog.Error("failed to subscribe",
			slog.String("error", err.Error()))
		timer := time.NewTimer(time.Second * 5)
		<-timer.C
		w.SubscribeUpdates(topic, token, id)
		return "", err
	}

	ctx := context.Background()
	for {
		msg, err := w.consumer.Receive(ctx)
		if err != nil {
			w.consumer.Close()

			timer := time.NewTimer(time.Second * 5)
			<-timer.C
			w.SubscribeUpdates(topic, token, id)
			return "", err
		} else if msg.Payload == nil || len(msg.Payload) == 0 {
			slog.Info("null message received", slog.String("msgId", msg.MsgId))
			w.consumer.Ack(context.Background(), msg)
			continue
		}

		_, ok := w.GetMessage(msg.Key)
		if !ok {
			w.SetMessage(msg.Key, msg)
		}

		var payloadType struct {
			AccountUid string                 `json:"account_uid"`
			Id         int                    `json:"Id"`
			Result     map[string]interface{} `json:"result"`
		}

		if msg == nil {
			err = w.consumer.Ack(context.Background(), msg)
			if err != nil {
				slog.Error("failed to ack the msg", "err", err)
				return "", err
			}
			continue
		}

		err = json.Unmarshal(msg.Payload, &payloadType)
		if err != nil {
			slog.Error("Synthetics [preview] failed to unmarshal payload", "err", err)
			return "", fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		if payloadType.Id == id {
			w.DeleteMessage(msg.Key)
			err = w.consumer.Ack(context.Background(), msg)
			if err != nil {
				slog.Error("failed to ack the matched data msg", "err", err)
				return "", fmt.Errorf("failed to ack the matched data msg: %w", err)
			}
			mapResult := payloadType.Result
			return mapResult["stdout"].(string), nil
		}

		err = w.consumer.Ack(context.Background(), msg)
		if err != nil {
			slog.Error("failed to ack the msg", "err", err)
			return "", fmt.Errorf("failed to ack the msg: %w", err)
		}
	}
}

func (w *Worker) produceMessage(accountUid string,
	topic string, key string, payload map[string]interface{}) {

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
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if w.cfg.NCAPassword != "" {
		req.Header.Set("Authorization", w.cfg.NCAPassword)
	}

	re, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to produce message", slog.String("error", err.Error()))
		return
	}

	if re.StatusCode != 200 {
		str, _ := io.ReadAll(re.Body)
		slog.Error("failed to produce message", slog.String("response status", re.Status),
			slog.String("url", url), slog.String("response", string(str)))
	}
}

func (w *Worker) sendRequest(accountUid string, id int, topic string, payload map[string]interface{}) {
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
	w.produceMessage(accountUid, topic, idString, payloadMap)
}
