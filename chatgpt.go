package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	buf "github.com/kklab-com/goth-bytebuf"
	"github.com/kklab-com/goth-kkutil/value"
)

const DefaultModel = "gpt-3.5-turbo"
const DefaultAPIEndpoint = "https://api.openai.com/v1/chat/completions"

type Opts struct {
	Model            string         `json:"model"`
	Temperature      *float32       `json:"temperature,omitempty"`
	TopP             *float32       `json:"top_p,omitempty"`
	N                *int           `json:"n,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	MaxTokens        *int           `json:"max_tokens,omitempty"`
	PresencePenalty  *float32       `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32       `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int `json:"logit_bias,omitempty"`
	User             string         `json:"user,omitempty"`
	//Stream           bool           `json:"stream,omitempty"`
}

type Request struct {
	Messages []Message `json:"messages"`
	Opts
}

type Response struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
		Index        int     `json:"index"`
	} `json:"choices"`
	Error *Error `json:"error"`
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

func (c *Client) say(messages []Message) (*Response, error) {
	url := c.ApiEndpoint
	header := http.Header{}
	header.Set("authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	header.Set("content-type", "application/json")
	header.Set("user-agent", "curl/7.79.1")
	header.Set("accept", "application/json")
	request, _ := http.NewRequest("POST", url, bytes.NewBufferString(value.JsonMarshal(Request{
		Messages: messages,
		Opts:     c.opts,
	})))

	request.Header = header
	httpResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	byteBuf := buf.NewByteBufString("").WriteReader(httpResponse.Body)
	response := &Response{}
	if err := json.Unmarshal(byteBuf.Bytes(), response); err != nil {
		return nil, err
	} else if response.Error != nil {
		return nil, response.Error
	} else {
		return response, nil
	}
}
