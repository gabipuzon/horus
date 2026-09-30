package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gabipuzon/horus/internal/incident"
)

type State string

const (
	Down           State = "DOWN"
	Recovered      State = "RECOVERED"
	requestTimeout       = 5 * time.Second
)

type Event struct {
	State       State
	MonitorName string
	MonitorURL  string
	Incident    incident.Incident
}

type Notifier interface {
	Notify(ctx context.Context, event Event) error
}

type Discord struct {
	webhookURL string
	client     *http.Client
	timeout    time.Duration
}

// HTTP errors may contain the webhook URL, including its secret token. Keep
// their log message safe while preserving cancellation checks through Unwrap.
type webhookRequestError struct {
	cause error
}

func (e *webhookRequestError) Error() string {
	switch {
	case errors.Is(e.cause, context.Canceled):
		return "Discord webhook request canceled"
	case errors.Is(e.cause, context.DeadlineExceeded):
		return "Discord webhook request timed out"
	default:
		return "Discord webhook request failed"
	}
}

func (e *webhookRequestError) Unwrap() error { return e.cause }

func NewDiscord(webhookURL string, client *http.Client) *Discord {
	if client == nil {
		client = &http.Client{}
	}
	return &Discord{webhookURL: webhookURL, client: client, timeout: requestTimeout}
}

type discordPayload struct {
	Content         string `json:"content"`
	AllowedMentions struct {
		Parse []string `json:"parse"`
	} `json:"allowed_mentions"`
}

func message(event Event) string {
	var text strings.Builder
	fmt.Fprintf(&text, "%s: %s\nURL: %s\nFailure: %s", event.State, event.MonitorName, event.MonitorURL, event.Incident.FailureType)
	if event.Incident.StatusCode != 0 {
		fmt.Fprintf(&text, "\nHTTP status: %d", event.Incident.StatusCode)
	}
	if event.Incident.FailureMessage != "" {
		fmt.Fprintf(&text, "\nError: %s", event.Incident.FailureMessage)
	}
	fmt.Fprintf(&text, "\nStarted: %s", event.Incident.StartedAt.UTC().Format(time.RFC3339))
	if event.State == Recovered && event.Incident.ResolvedAt != nil {
		resolved := *event.Incident.ResolvedAt
		duration := resolved.Sub(event.Incident.StartedAt)
		if duration < 0 {
			duration = 0
		}
		fmt.Fprintf(&text, "\nRecovered: %s\nDuration: %s", resolved.UTC().Format(time.RFC3339), duration.Round(time.Second))
	}
	return text.String()
}

func (d *Discord) Notify(ctx context.Context, event Event) error {
	body := discordPayload{Content: message(event)}
	body.AllowedMentions.Parse = []string{}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	requestCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, d.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return &webhookRequestError{cause: err}
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := d.client.Do(req)
	if err != nil {
		return &webhookRequestError{cause: err}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Discord webhook returned HTTP %d", response.StatusCode)
	}
	return nil
}
