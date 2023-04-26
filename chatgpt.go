package chatgpt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	buf "github.com/kklab-com/goth-bytebuf"
	"github.com/kklab-com/goth-kkutil/value"
)

const (
	GPT3_5_TURBO = "gpt-3.5-turbo"
	GPT4         = "gpt-4"
	GPT4_32K     = "gpt-4-32k"
	DefaultModel = GPT3_5_TURBO
)
const DefaultAPIEndpoint = "https://api.openai.com/v1/chat/completions"

type FinishReason string

const (
	FinishReasonStop   FinishReason = "stop"
	FinishReasonLength              = "length"
)

type Opts struct {
	Model            string         `json:"model"`
	Temperature      *float32       `json:"temperature,omitempty"`
	TopP             *float32       `json:"top_p,omitempty"`
	N                int            `json:"n,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	PresencePenalty  float32        `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32        `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int `json:"logit_bias,omitempty"`
	User             string         `json:"user,omitempty"`
}

type Request struct {
	Messages []Message `json:"messages"`
	Opts
	Stream bool `json:"stream,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Choice struct {
	Message      Message      `json:"message"`
	FinishReason FinishReason `json:"finish_reason"`
	Index        int          `json:"index"`
}

type Response struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Model   string   `json:"model"`
	Usage   Usage    `json:"usage"`
	Choices []Choice `json:"choices"`
	Error   *Error   `json:"error"`
}

type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

func (e *Error) Error() string {
	return value.JsonMarshal(e)
}

type Role string

const (
	System    Role = "system"
	User           = "user"
	Assistant      = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type Client struct {
	ApiEndpoint string
	apiKey      string
	opts        Opts
}

func NewClient(apiKey string) *Client {
	return NewClientWithOpts(apiKey, Opts{Model: DefaultModel})
}

func NewClientWithOpts(apiKey string, opts Opts) *Client {
	if apiKey == "" {
		return nil
	}

	if opts.Model == "" {
		opts.Model = DefaultModel
	}

	return &Client{apiKey: apiKey, ApiEndpoint: DefaultAPIEndpoint, opts: opts}
}

func (c *Client) NewThread() *Thread {
	return &Thread{client: *c}
}

func (c *Client) Opts() *Opts {
	return &c.opts
}

func (c *Client) Say(messages []Message) (*Response, error) {
	request, _ := http.NewRequest("POST", c.ApiEndpoint, bytes.NewBufferString(value.JsonMarshal(Request{
		Messages: messages,
		Opts:     c.opts,
	})))

	request.Header = http.Header{}
	request.Header.Set("authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	request.Header.Set("content-type", "application/json")
	request.Header.Set("user-agent", "curl/7.79.1")
	request.Header.Set("accept", "application/json")
	httpResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	byteBuf := buf.EmptyByteBuf().WriteReader(httpResponse.Body)
	httpResponse.Body.Close()
	response := &Response{}
	if err := json.Unmarshal(byteBuf.Bytes(), response); err != nil {
		return nil, err
	} else if response.Error != nil {
		return nil, response.Error
	} else {
		return response, nil
	}
}

type StreamEvent struct {
	Event string     `json:"event,omitempty"`
	Id    string     `json:"id,omitempty"`
	Data  StreamData `json:"data,omitempty"`
}

type StreamChoice struct {
	Delta        StreamDelta  `json:"delta"`
	Index        int          `json:"index"`
	FinishReason FinishReason `json:"finish_reason"`
}

type StreamDelta struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type StreamData struct {
	Id      string         `json:"id"`
	Object  string         `json:"object"`
	Created int            `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Done    bool           `json:"-"`
}

type StreamScanner struct {
	closed bool
	rc     io.ReadCloser
	sc     *bufio.Scanner
	Role   Role
}

func (d *StreamScanner) Next() (*StreamEvent, error) {
	var event *StreamEvent
	var dataString = ""
	for d.sc.Scan() {
		line := d.sc.Text()
		if event == nil {
			event = &StreamEvent{}
		}

		if line == "" {
			if event.Data.Done {
				return event, nil
			}

			if err := json.Unmarshal([]byte(dataString), &event.Data); err != nil {
				return nil, err
			}

			return event, nil
		}

		parts := strings.SplitN(line, ":", 2)
		switch parts[0] {
		case "event":
			event.Event = parts[1]
		case "id":
			event.Id = parts[1]
		case "data":
			if parts[1] == " [DONE]" {
				event.Data.Done = true
			} else {
				dataString += parts[1]
			}
		}
	}

	if !d.closed {
		d.closed = true
		defer d.rc.Close()
	}

	if err := d.sc.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}

func (c *Client) Stream(messages []Message) (*StreamScanner, error) {
	opts := c.opts
	request, _ := http.NewRequest("POST", c.ApiEndpoint, bytes.NewBufferString(value.JsonMarshal(Request{
		Messages: messages,
		Opts:     opts,
		Stream:   true,
	})))

	request.Header = http.Header{}
	request.Header.Set("authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	request.Header.Set("content-type", "application/json")
	request.Header.Set("user-agent", "curl/7.79.1")
	request.Header.Set("accept", "application/json")
	httpResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	return &StreamScanner{
		rc: httpResponse.Body,
		sc: bufio.NewScanner(httpResponse.Body),
	}, nil
}
