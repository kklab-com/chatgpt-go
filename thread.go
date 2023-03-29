package chatgpt

import "io"

type Thread struct {
	client   Client
	Messages []Message
}

func (t *Thread) Client() *Client {
	return &t.client
}

func (t *Thread) Behavior(content string) *Thread {
	if content == "" {
		if len(t.Messages) > 0 && t.Messages[0].Role == System {
			t.Messages = t.Messages[1:]
			return t
		}
	}

	if len(t.Messages) == 0 {
		t.Messages = append(t.Messages, Message{
			Role:    System,
			Content: content,
		})
	} else {
		if t.Messages[0].Role == System {
			t.Messages[0].Content = content
		} else {
			t.Messages = append([]Message{{Role: System, Content: content}}, t.Messages...)
		}
	}

	return t
}

func (t *Thread) Say(content string) (*Response, error) {
	t.Messages = append(t.Messages, Message{
		Role:    User,
		Content: content,
	})

	if response, err := t.client.Say(t.Messages); err != nil {
		t.Messages = t.Messages[:len(t.Messages)-1]
		return nil, err
	} else {
		t.Messages = append(t.Messages, response.Choices[0].Message)
		return response, nil
	}
}

func (t *Thread) Talk(content string) *Response {
	t.Messages = append(t.Messages, Message{
		Role:    User,
		Content: content,
	})

	if response, err := t.client.Say(t.Messages); err != nil {
		t.Messages = t.Messages[:len(t.Messages)-1]
		switch cast := err.(type) {
		case *Error:
			if cast.Code == "context_length_exceeded" {
				if l := len(t.Messages); l == 1 {
					panic(cast)
				} else {
					if t.Messages[0].Role == System {
						if l > 2 {
							idx := 0
							for i, message := range t.Messages {
								if message.Role == Assistant {
									if i < l-1 {
										idx = i
										break
									}
								}
							}

							if idx == 0 {
								t.Messages = []Message{t.Messages[0]}
							} else {
								t.Messages = append([]Message{t.Messages[0]}, t.Messages[idx+1:]...)
							}
						} else {
							t.Messages = []Message{t.Messages[0]}
						}
					} else {
						idx := 0
						for i, message := range t.Messages {
							if message.Role == Assistant {
								if i < l-1 {
									idx = i
								}
							}
						}

						if idx == 0 {
							t.Messages = nil
						} else {
							t.Messages = t.Messages[idx+1:]
						}
					}

					return t.Talk(content)
				}
			} else {
				return &Response{Error: cast}
			}
		}

		return nil
	} else {
		t.Messages = append(t.Messages, response.Choices[0].Message)
		return response
	}
}

func (t *Thread) Stream(content string) (*ThreadStreamScanner, error) {
	t.Messages = append(t.Messages, Message{
		Role:    User,
		Content: content,
	})

	if decoder, err := t.client.Stream(t.Messages); err != nil {
		t.Messages = t.Messages[:len(t.Messages)-1]
		return nil, err
	} else {
		return &ThreadStreamScanner{t: t, sd: decoder}, nil
	}
}

type ThreadStreamScanner struct {
	t            *Thread
	Id           string `json:"id"`
	Object       string `json:"object"`
	Created      int    `json:"created"`
	Model        string `json:"model"`
	Role         Role
	Content      string
	FinishReason FinishReason
	Done         bool
	sd           *StreamScanner
}

func (d *ThreadStreamScanner) Next() (*StreamEvent, error) {
	if evt, err := d.sd.Next(); err == nil {
		if evt.Data.Done {
			d.Done = true
			d.t.Messages = append(d.t.Messages, Message{Role: d.Role, Content: d.Content})
			return nil, err
		}

		if v := evt.Data.Id; d.Id == "" && v != "" {
			d.Id = v
		}

		if v := evt.Data.Object; d.Object == "" && v != "" {
			d.Object = v
		}

		if v := evt.Data.Created; d.Created == 0 && v != 0 {
			d.Created = v
		}

		if v := evt.Data.Model; d.Model == "" && v != "" {
			d.Model = v
		}

		if v := evt.Data.Choices[0].Delta.Role; d.Role == "" && v != "" {
			d.Role = v
			return d.Next()
		}

		if v := evt.Data.Choices[0].Delta.Content; v != "" {
			d.Content += v
		}

		if v := evt.Data.Choices[0].FinishReason; v != "" {
			d.FinishReason = v
		}

		return evt, err
	} else {
		if err == io.EOF {
			return nil, nil
		} else {
			d.t.Messages = d.t.Messages[:len(d.t.Messages)-1]
			return nil, err
		}
	}
}

func (d *ThreadStreamScanner) FetchAll() error {
	for !d.Done {
		next, err := d.Next()
		if err != nil {
			return err
		} else if next != nil {
			continue
		}
	}

	return nil
}
