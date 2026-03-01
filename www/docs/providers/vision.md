# Vision (Multimodal Images)

Send images to LLMs for analysis using URL references or raw binary data. Works with any provider that supports multimodal input (Anthropic, OpenAI, Gemini).

## Image from URL

```go
import "github.com/joakimcarlsson/ai/message"

msg := message.NewUserMessage("What do you see in this image?")
msg.AddImageURL("https://example.com/photo.jpg", "")

response, err := client.SendMessages(ctx, []message.Message{msg}, nil)
fmt.Println(response.Content)
```

The second argument to `AddImageURL` is an optional detail level (`"low"`, `"high"`, or `""` for auto).

## Image from Binary Data

```go
imageData, _ := os.ReadFile("photo.jpg")

msg := message.NewUserMessage("Describe this image.")
msg.AddBinary("image/jpeg", imageData)

response, err := client.SendMessages(ctx, []message.Message{msg}, nil)
fmt.Println(response.Content)
```

## Multiple Images

```go
msg := message.NewUserMessage("Compare these two images.")
msg.AddImageURL("https://example.com/before.jpg", "")
msg.AddImageURL("https://example.com/after.jpg", "")

response, err := client.SendMessages(ctx, []message.Message{msg}, nil)
```

## MultiModalMessage

For full control, build messages with the `MultiModalMessage` type directly:

```go
msg := message.NewUserMultiModalMessage([]message.MultiModalContent{
    message.NewTextContent("What's in this image?"),
    message.NewImageURLContent("https://example.com/photo.jpg", "high"),
})

// Or with attachments
msg := message.NewUserMultiModalMessageWithAttachments(
    "Describe these files.",
    []message.Attachment{
        {MIMEType: "image/png", Data: pngData},
        {MIMEType: "image/jpeg", Data: jpegData},
    },
)
```

## Content Types

| Type | Constructor | Description |
|------|-------------|-------------|
| `text` | `NewTextContent(text)` | Text content |
| `image_url` | `NewImageURLContent(url, detail)` | Image from URL |
| `binary` | `NewBinaryContent(mimeType, data)` | Raw binary data (base64-encoded for the provider) |

## Supported Formats

Most providers accept JPEG, PNG, GIF, and WebP. Check your provider's documentation for size limits.
