package lumex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultAPIHost is the public Bot API endpoint.
const DefaultAPIHost = "https://api.telegram.org"

// Marshaler and Unmarshaler mirror encoding/json's function signatures, so any
// drop-in codec (json-iterator, goccy/go-json, sonic, …) can be plugged in.
// A replacement MUST still honour the json.Marshaler / json.Unmarshaler
// interfaces implemented by the generated types (discriminators, InputFile, …).
type (
	Marshaler   func(v any) ([]byte, error)
	Unmarshaler func(data []byte, v any) error
)

// BotClient talks to the Telegram Bot API.
type BotClient interface {
	// RequestWithContext submits a method call with its request payload and
	// returns the raw JSON `result` (or a *TelegramError on ok:false).
	RequestWithContext(ctx context.Context, method string, request any) (json.RawMessage, error)
}

var _ BotClient = (*Client)(nil)

// Uploadable is implemented (via generated methods) by request types that can
// carry files — directly or nested in media. The client builds the multipart
// body from it without reflection or a marshal→unmarshal round-trip:
//
//   - Fields returns all parameters honouring omitempty; file fields marshal to
//     "attach://<id>" once ids are assigned;
//   - Files returns every uploadable *InputFile (including ones nested in media).
type Uploadable interface {
	Fields() map[string]any
	Files() []*InputFile
}

// ClientOption configures a Client; pass any number to NewClient.
type ClientOption func(*Client)

// WithAPIHost overrides the Bot API base URL (e.g. a self-hosted Bot API server).
func WithAPIHost(host string) ClientOption {
	return func(c *Client) { c.apiHost = strings.TrimSuffix(host, "/") }
}

// WithHTTPClient sets the underlying HTTP client (pooling, proxies, timeouts…).
func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *Client) {
		if h != nil {
			c.http = h
		}
	}
}

// WithMarshaler overrides the request codec (e.g. sonic.Marshal).
func WithMarshaler(m Marshaler) ClientOption {
	return func(c *Client) {
		if m != nil {
			c.marshal = m
		}
	}
}

// WithUnmarshaler overrides the response codec (e.g. sonic.Unmarshal).
func WithUnmarshaler(u Unmarshaler) ClientOption {
	return func(c *Client) {
		if u != nil {
			c.unmarshal = u
		}
	}
}

// Client is the default BotClient implementation.
type Client struct {
	token     string
	apiHost   string
	http      *http.Client
	marshal   Marshaler
	unmarshal Unmarshaler
}

// NewClient builds a Client for the given bot token. With no options it uses
// sensible defaults: the public API host, encoding/json, and a fresh
// *http.Client.
func NewClient(token string, opts ...ClientOption) *Client {
	c := &Client{
		token:     token,
		apiHost:   DefaultAPIHost,
		http:      &http.Client{},
		marshal:   json.Marshal,
		unmarshal: json.Unmarshal,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Client) methodURL(method string) string {
	return c.apiHost + "/bot" + c.token + "/" + method
}

// FileURL builds the download URL for a file path returned by getFile.
func (c *Client) FileURL(filePath string) string {
	return c.apiHost + "/file/bot" + c.token + "/" + filePath
}

// Response is the Telegram API envelope wrapping every reply.
type Response struct {
	OK          bool                `json:"ok"`
	Result      json.RawMessage     `json:"result"`
	ErrorCode   int                 `json:"error_code"`
	Description string              `json:"description"`
	Parameters  *ResponseParameters `json:"parameters"`
}

// TelegramError is returned when the API responds with ok:false.
type TelegramError struct {
	// Method is the Bot API method that failed.
	Method string
	// Code is the numeric error code (roughly HTTP-like).
	Code int
	// Description is the human-readable error message.
	Description string
	// Parameters carries optional recovery hints (retry_after, migrate_to_chat_id).
	Parameters *ResponseParameters
}

func (e *TelegramError) Error() string {
	return fmt.Sprintf("%s: %d %s", e.Method, e.Code, e.Description)
}

// RetryAfter reports the flood-control delay Telegram asks the caller to wait,
// or 0 if none was provided.
func (e *TelegramError) RetryAfter() time.Duration {
	if e.Parameters != nil && e.Parameters.RetryAfter > 0 {
		return time.Duration(e.Parameters.RetryAfter) * time.Second
	}
	return 0
}

// MigrateToChatID reports the new chat id when a group migrated to a supergroup.
func (e *TelegramError) MigrateToChatID() (int64, bool) {
	if e.Parameters != nil && e.Parameters.MigrateToChatID != 0 {
		return e.Parameters.MigrateToChatID, true
	}
	return 0, false
}

// RequestWithContext encodes the request (JSON, or multipart when it carries
// files to upload), performs the POST, and returns the raw `result`.
func (c *Client) RequestWithContext(ctx context.Context, method string, request any) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	body, contentType, err := c.encode(request)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", method, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.methodURL(method), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	// A replayable body lets the HTTP client retry on HTTP/2 GO_AWAY.
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, sanitizeError(c.token, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", method, err)
	}
	var r Response
	if err := c.unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", method, err)
	}
	if !r.OK {
		return nil, &TelegramError{Method: method, Code: r.ErrorCode, Description: r.Description, Parameters: r.Parameters}
	}
	return r.Result, nil
}

// encode returns the request body and its content type. It takes the JSON
// fast-path (a single struct marshal) unless the request is Uploadable and
// actually carries a file to upload.
func (c *Client) encode(request any) ([]byte, string, error) {
	if u, ok := request.(Uploadable); ok {
		var uploads []*InputFile
		for _, f := range u.Files() {
			if f != nil && f.NeedsUpload() {
				uploads = append(uploads, f)
			}
		}
		if len(uploads) > 0 {
			// Assign each upload a unique multipart part id so InputFile's
			// MarshalJSON emits a matching "attach://fileN".
			for i, f := range uploads {
				f.attachID = "file" + strconv.Itoa(i)
			}
			return c.encodeMultipart(u, uploads)
		}
	}
	b, err := c.marshalRequest(request)
	if err != nil {
		return nil, "", err
	}
	return b, "application/json", nil
}

// encodeMultipart builds a multipart/form-data body from the request's Fields
// (each file field already renders to "attach://fileN") plus a file part per
// upload, named after its assigned attach id.
func (c *Client) encodeMultipart(u Uploadable, uploads []*InputFile) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for k, v := range u.Fields() {
		s, err := c.fieldValue(v)
		if err != nil {
			return nil, "", err
		}
		if err := w.WriteField(k, s); err != nil {
			return nil, "", err
		}
	}

	for _, f := range uploads {
		filename := f.Name
		if filename == "" {
			filename = f.attachID
		}
		part, err := w.CreateFormFile(f.attachID, filename)
		if err != nil {
			return nil, "", err
		}
		if _, err := io.Copy(part, f.Reader); err != nil {
			return nil, "", err
		}
	}

	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.FormDataContentType(), nil
}

func (c *Client) marshalRequest(request any) ([]byte, error) {
	if request == nil {
		return []byte("{}"), nil
	}
	return c.marshal(request)
}

// fieldValue renders a single Fields() value as a multipart form field. Scalars
// take a reflection-free fast path (strconv, no json.Marshal); everything else
// (ChatID, keyboards, entities, unions, …) is encoded with the client codec so
// its MarshalJSON semantics are preserved.
func (c *Client) fieldValue(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case bool:
		if val {
			return "true", nil
		}
		return "false", nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case int:
		return strconv.Itoa(val), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	default:
		b, err := c.marshal(val)
		if err != nil {
			return "", err
		}
		return formValue(b), nil
	}
}

// formValue unwraps a JSON value for use as a form field: JSON strings are sent
// unquoted, everything else as its raw JSON.
func formValue(v []byte) string {
	if len(v) > 0 && v[0] == '"' {
		var s string
		if json.Unmarshal(v, &s) == nil {
			return s
		}
	}
	return string(v)
}

// sanitizeError scrubs the bot token from URL errors so it never reaches logs.
func sanitizeError(token string, err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		ue.URL = strings.ReplaceAll(ue.URL, token, "<token>")
		return ue
	}
	return err
}

// Call is a generic convenience wrapper that decodes the raw result into T.
//
//	me, err := telegram.Call[telegram.User](ctx, c, "getMe", telegram.GetMeRequest{})
func Call[T any](ctx context.Context, c BotClient, method string, request any) (T, error) {
	var out T
	raw, err := c.RequestWithContext(ctx, method, request)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("decode %s result: %w", method, err)
	}
	return out, nil
}
