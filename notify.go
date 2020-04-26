// Package linenotify provides notify function by LINE
package linenotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/xerrors"
)

const notifyURL = "https://notify-api.line.me/api/notify"

type NotifyResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (r *NotifyResponse) Error() string {
	jsonStr, _ := json.Marshal(r)
	return string(jsonStr)
}

type Notifier interface {
	Send(ctx context.Context, message string) error
}

type notifier struct {
	endpoint  *url.URL
	authToken string
	client    *http.Client
}

func New(token string) (Notifier, error) {
	if token == "" {
		return nil, xerrors.Errorf("authorization token is not set")
	}

	endpoint, _ := url.Parse(notifyURL)

	return &notifier{
		endpoint:  endpoint,
		authToken: token,
		client:    http.DefaultClient,
	}, nil
}

// https://notify-bot.line.me/doc/ja/
func (n *notifier) Send(ctx context.Context, message string) error {
	body := url.Values{}
	body.Set("message", message)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		n.endpoint.String(),
		strings.NewReader(body.Encode()),
	)
	if err != nil {
		return xerrors.Errorf("Cannot initialize HTTP request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+n.authToken)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return xerrors.Errorf("Failed to send request: %w", err)
	}

	return checkResponse(response)
}

func checkResponse(response *http.Response) error {
	notifyResponse := NotifyResponse{}

	err := json.NewDecoder(response.Body).Decode(&notifyResponse)
	if err != nil {
		return xerrors.Errorf("Failed to read response body: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return &notifyResponse
	}

	return nil
}
