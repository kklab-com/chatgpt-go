# ChatGPT-go

golang version ChatGPT API

## HOWTO

```golang
// create client
client := chatgpt.NewClient(OPENAI_API_KEY)

// create thread, thread can remember every interaction.
thread := client.NewThread()

// interactive with error response
thread.Say("something")

// fast interactive and auto prompt length exceed fix
thread.Talk("something")

```