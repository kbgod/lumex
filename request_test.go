package lumex

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelegramError_Error(t *testing.T) {
	method := "sendMessage"
	params := map[string]any{"chat_id": "12345", "text": "Hello"}
	code := 400
	description := "Bad Request: chat not found"
	responseParams := &ResponseParameters{MigrateToChatId: 67890, RetryAfter: 5}

	telegramError := &TelegramError{
		Method:         method,
		Params:         params,
		Code:           code,
		Description:    description,
		ResponseParams: responseParams,
	}

	errorMessage := telegramError.Error()

	expectedMessage := "unable to sendMessage: Bad Request: chat not found"
	if errorMessage != expectedMessage {
		t.Errorf("unexpected error message: got %q, want %q", errorMessage, expectedMessage)
	}

	if telegramError.Method != method {
		t.Errorf("unexpected Method: got %q, want %q", telegramError.Method, method)
	}

	if telegramError.Code != code {
		t.Errorf("unexpected Code: got %d, want %d", telegramError.Code, code)
	}

	if telegramError.Description != description {
		t.Errorf("unexpected Description: got %q, want %q", telegramError.Description, description)
	}

	if telegramError.ResponseParams.MigrateToChatId != 67890 {
		t.Errorf("unexpected MigrateToChatID: got %d, want %d", telegramError.ResponseParams.MigrateToChatId, 67890)
	}

	if telegramError.ResponseParams.RetryAfter != 5 {
		t.Errorf("unexpected RetryAfter: got %d, want %d", telegramError.ResponseParams.RetryAfter, 5)
	}
}

func TestBaseBotClient_RequestWithContext(t *testing.T) {
	if os.Getenv("TEST_BOT_TOKEN") == "" {
		t.Skip()
	}

	client, err := NewBot(os.Getenv("TEST_BOT_TOKEN"), nil)
	assert.NoError(t, err, "NewBot() failed")

	t.Run("without parameters", func(t *testing.T) {
		resp, err := client.RequestWithContext(nil, "getMe", nil, nil)
		assert.NoError(t, err, "RequestWithContext() failed")
		assert.NotNil(t, resp, "Response is nil")

		var u User
		err = json.Unmarshal(resp, &u)
		assert.NoError(t, err, "json.Unmarshal() failed")

		assert.Equal(t, "lumex_test_bot", u.Username, "unexpected username")
	})

	t.Run("file by url", func(t *testing.T) {
		v := map[string]any{
			"chat_id": 1681111384,
		}

		v["photo"] = InputFileByURL("https://telegram.org/img/t_logo.png")

		resp, err := client.RequestWithContext(context.Background(), "sendPhoto", v, nil)
		assert.NoError(t, err, "RequestWithContext() failed")
		assert.NotNil(t, resp, "Response is nil")
	})

	t.Run("file by reader", func(t *testing.T) {
		v := map[string]any{
			"chat_id": "1681111384",
		}

		photo, err := http.Get("https://telegram.org/img/t_logo.png")
		assert.NoError(t, err, "http.Get() failed")

		v["photo"] = InputFileByReader("photo", photo.Body)

		resp, err := client.RequestWithContext(context.Background(), "sendPhoto", v, nil)
		assert.NoError(t, err, "RequestWithContext() failed")
		assert.NotNil(t, resp, "Response is nil")
	})

	t.Run("telegram error", func(t *testing.T) {
		v := map[string]any{
			"chat_id": "1",
			"text":    "Hello",
		}

		_, err := client.RequestWithContext(context.Background(), "sendMessage", v, nil)
		assert.Error(t, err, "RequestWithContext() should fail")
		assert.IsType(t, &TelegramError{}, err, "unexpected error type")
	})
}

func TestBaseBotClient_GetAPIURL(t *testing.T) {
	tests := []struct {
		name           string
		opts           *RequestOpts
		defaultRequest *RequestOpts
		expectedAPIURL string
	}{
		{
			name:           "RequestOpts has APIURL",
			opts:           &RequestOpts{APIURL: "https://custom.api.com/"},
			defaultRequest: nil,
			expectedAPIURL: "https://custom.api.com",
		},
		{
			name:           "DefaultRequestOpts has APIURL",
			opts:           nil,
			defaultRequest: &RequestOpts{APIURL: "https://default.api.com/"},
			expectedAPIURL: "https://default.api.com",
		},
		{
			name:           "Both RequestOpts and DefaultRequestOpts are nil",
			opts:           nil,
			defaultRequest: nil,
			expectedAPIURL: DefaultAPIURL,
		},
		{
			name:           "RequestOpts APIURL is empty, use DefaultRequestOpts",
			opts:           &RequestOpts{APIURL: ""},
			defaultRequest: &RequestOpts{APIURL: "https://default.api.com/"},
			expectedAPIURL: "https://default.api.com",
		},
		{
			name:           "Both RequestOpts and DefaultRequestOpts APIURL are empty",
			opts:           &RequestOpts{APIURL: ""},
			defaultRequest: &RequestOpts{APIURL: ""},
			expectedAPIURL: DefaultAPIURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bot := &BaseBotClient{
				DefaultRequestOpts: tt.defaultRequest,
			}
			got := bot.GetAPIURL(tt.opts)
			if got != tt.expectedAPIURL {
				t.Errorf("GetAPIURL() = %v, want %v", got, tt.expectedAPIURL)
			}
		})
	}
}

func TestBaseBotClient_getEnvAuth(t *testing.T) {

	t.Run("test env", func(t *testing.T) {
		client := BaseBotClient{
			UseTestEnvironment: true,
		}

		auth := client.getEnvAuth("123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11")

		assert.Equal(t, "bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/test", auth, "unexpected auth")
	})

	t.Run("production env", func(t *testing.T) {
		client := BaseBotClient{
			UseTestEnvironment: false,
		}

		auth := client.getEnvAuth("123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11")

		assert.Equal(t, "bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", auth, "unexpected auth")
	})
}

func TestBaseBotClient_FileURL(t *testing.T) {
	client := BaseBotClient{
		UseTestEnvironment: false,
	}

	fileURL := client.FileURL("123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", "photos/file_123.jpg", nil)

	assert.Equal(
		t,
		"https://api.telegram.org/file/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/photos/file_123.jpg",
		fileURL,
		"unexpected file URL",
	)
}
