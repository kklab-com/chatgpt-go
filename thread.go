package chatgpt

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

	if response, err := t.client.say(t.Messages); err != nil {
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

	if response, err := t.client.say(t.Messages); err != nil {
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
