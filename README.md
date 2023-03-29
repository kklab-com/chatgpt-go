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

// server-sent event, streaming interaction
if stream, err := thread.Stream("how are you"); err == nil {
    for {
        next, err := stream.Next()
        if err != nil {
          println(err.Error())
        } else if next != nil {
          println(next.Data.Choices[0].Delta.Content)
        } else if stream.Done {
          break
        }
    }
    
    // server-sent event, fetch all until ` [DONE]`
    if err := stream.FetchAll(); err != nil {
      println(err.Error())
    }
    
    // print all content of this streaming session
    println(stream.Content)
}

```