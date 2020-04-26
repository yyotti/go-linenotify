package linenotify

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
)

func TestNew(t *testing.T) {
	tests := []struct {
		token string

		expectedError error
	}{
		{
			token: "AABBCCDDEE",
		},
		{
			token:         "",
			expectedError: xerrors.New("authorization token is not set"),
		},
	}

	for i, tt := range tests {
		tt := tt // capture

		a := assert.New(t)

		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			t.Parallel()

			actual, err := New(tt.token)
			if tt.expectedError != nil {
				a.NotNil(err)
				a.EqualError(err, tt.expectedError.Error())
			} else {
				a.Nil(err)
				a.NotNil(actual)
			}
		})
	}
}

const messageOK = `{"status":200,"message":"ok"}`

func writeNotifyError(w http.ResponseWriter, status int, message string, msgArgs ...interface{}) {
	w.WriteHeader(status)

	msg := fmt.Sprintf(message, msgArgs...)
	if _, err := fmt.Fprintf(w, `{"status":%d,"message":"%s"}`, status, msg); err != nil {
		panic(err)
	}
}

func testHandler(token string, message string, resStatus int, resBody string) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				writeNotifyError(
					w, http.StatusBadRequest,
					"Unexpected request: method = %s", r.Method,
				)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer "+token {
				writeNotifyError(
					w, http.StatusBadRequest,
					"Unexpected `Authorization` header: %s", authHeader,
				)
				return
			}

			contentTypeHeader := r.Header.Get("Content-Type")
			if contentTypeHeader != "application/x-www-form-urlencoded" {
				writeNotifyError(
					w, http.StatusBadRequest,
					"Unexpected `Content-Type` header: %s", contentTypeHeader,
				)
				return
			}

			err := r.ParseForm()
			if err != nil {
				writeNotifyError(
					w, http.StatusBadRequest,
					"Failed to parse form: %v", err,
				)
			}

			messageParam := r.PostForm.Get("message")
			if messageParam != message {
				writeNotifyError(
					w, http.StatusBadRequest,
					"Unexpected message: %s", messageParam,
				)
				return
			}

			if resStatus == http.StatusOK {
				w.WriteHeader(resStatus)

				if _, err := w.Write([]byte(messageOK)); err != nil {
					log.Fatal(err)
				}

				return
			}

			writeNotifyError(w, resStatus, resBody)
		},
	)
}

func TestNotifier_Send(t *testing.T) {
	tests := []struct {
		token   string
		message string

		resStatus int
		resBody   string

		expectedError error
	}{
		{
			token:     "AABBCCDDEE",
			message:   "This is TEST message.",
			resStatus: http.StatusOK,
		},
	}

	for i, tt := range tests {
		tt := tt // capture

		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				testHandler(tt.token, tt.message, tt.resStatus, tt.resBody),
			)
			defer server.Close()

			testURL, _ := url.Parse(server.URL)

			n := &notifier{
				endpoint:  testURL,
				authToken: tt.token,
				client:    server.Client(),
			}
			err := n.Send(context.Background(), tt.message)
			if tt.expectedError != nil {
				assert.NotNil(t, err)
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCheckResponse(t *testing.T) {
	var tests = []struct {
		status   int
		response string

		expectError   bool
		expectedError error
	}{
		{
			status:      http.StatusOK,
			response:    `{"status":200,"message":"ok"}`,
			expectError: false,
		},
		{
			status:      http.StatusNotFound,
			response:    `{"status":404,"message":"not found"}`,
			expectError: true,
			expectedError: &NotifyResponse{
				Status:  404,
				Message: "not found",
			},
		},
		{
			status:      http.StatusOK,
			response:    `{status:200,"message":"ok"}`, // Invalid JSON format
			expectError: true,
		},
	}

	for i, tt := range tests {
		tt := tt

		a := assert.New(t)
		t.Run(fmt.Sprintf("%02d", i), func(*testing.T) {
			actual := checkResponse(tt.status, []byte(tt.response))

			if tt.expectError {
				a.NotNil(actual)
				if tt.expectedError != nil {
					a.EqualError(actual, tt.expectedError.Error())
				}
			} else {
				a.Nil(actual)
			}
		})
	}
}
