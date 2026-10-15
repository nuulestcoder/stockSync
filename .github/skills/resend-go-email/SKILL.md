---
name: resend-go-email
description: Implement and review email-sending functionality in Go using the official Resend Go SDK. Use this skill when adding transactional emails, email notifications, HTML or text email delivery, CC/BCC, reply-to addresses, attachments, templates, or Resend integration.
---

# Resend email sending with Go

Use the official Resend Go SDK:

```bash
go get github.com/resend/resend-go/v4
```

## Core requirements

When implementing Resend email delivery:

1. Read the existing project structure, Go version, configuration conventions, and dependency management before editing files.
2. Use `github.com/resend/resend-go/v4`.
3. Never hard-code the Resend API key.
4. Read the API key from an environment variable, preferably `RESEND_API_KEY`.
5. Do not log the API key, email body, or sensitive recipient information.
6. Validate required input:
   - sender address
   - at least one recipient
   - subject
   - HTML or plain-text content
7. Use context-aware methods when the application already uses `context.Context`.
8. Return useful errors to the caller instead of silently ignoring send failures.
9. Add tests using a mocked HTTP server or an injectable mail-sending interface.
10. Do not send a real email during tests unless the user explicitly requests it.
11. Before performing a real send in a development or debugging task, ask for confirmation unless the user explicitly requested the send.
12. Use a verified sender domain in production. The `onboarding@resend.dev` sender is suitable for basic testing only.

## Basic implementation

Use this pattern for a minimal email sender:

```go
package email

import (
	"context"
	"fmt"
	"os"

	"github.com/resend/resend-go/v4"
)

type Sender struct {
	client *resend.Client
}

func NewSender() (*Sender, error) {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY is not set")
	}

	return &Sender{
		client: resend.NewClient(apiKey),
	}, nil
}

type Message struct {
	From    string
	To      []string
	Subject string
	HTML    string
	Text    string
	Cc      []string
	Bcc     []string
	ReplyTo string
}

func (s *Sender) Send(ctx context.Context, message Message) (string, error) {
	if message.From == "" {
		return "", fmt.Errorf("from address is required")
	}
	if len(message.To) == 0 {
		return "", fmt.Errorf("at least one recipient is required")
	}
	if message.Subject == "" {
		return "", fmt.Errorf("subject is required")
	}
	if message.HTML == "" && message.Text == "" {
		return "", fmt.Errorf("HTML or text content is required")
	}

	params := &resend.SendEmailRequest{
		From:    message.From,
		To:      message.To,
		Subject: message.Subject,
		Html:    message.HTML,
		Text:    message.Text,
		Cc:      message.Cc,
		Bcc:     message.Bcc,
		ReplyTo: message.ReplyTo,
	}

	sent, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return "", fmt.Errorf("send email with Resend: %w", err)
	}

	return sent.Id, nil
}
```

If the surrounding code does not use contexts, `client.Emails.Send(params)` is acceptable for a simple integration. Prefer `SendWithContext` for HTTP handlers, workers, jobs, and services.

## Configuration

Use environment variables or the application's existing configuration system:

```bash
export RESEND_API_KEY="re_..."
```

For local development, use a `.env` file only if the project already supports one. Never commit real API keys. Ensure `.env` is listed in `.gitignore`.

## Message construction

For HTML email:

```go
params := &resend.SendEmailRequest{
	From:    "Acme <onboarding@resend.dev>",
	To:      []string{"delivered@resend.dev"},
	Subject: "Hello from Golang",
	Html:    "<strong>hello world</strong>",
}
```

For a multipart-friendly email, provide both `Html` and `Text`:

```go
params := &resend.SendEmailRequest{
	From:    "Acme <sender@example.com>",
	To:      []string{"customer@example.com"},
	Subject: "Your notification",
	Html:    "<p>Your notification content.</p>",
	Text:    "Your notification content.",
}
```

Optional fields include:

- `Cc`
- `Bcc`
- `ReplyTo`
- `Attachments`
- `Headers`
- `Tags`
- `ScheduledAt`
- `Template`

Use the SDK's current types and method signatures rather than inventing custom request fields.

## Error handling

Wrap errors with operation context:

```go
sent, err := client.Emails.Send(params)
if err != nil {
	return fmt.Errorf("send welcome email: %w", err)
}
```

Do not print only `err.Error()` in reusable packages. Return the error to the caller so the application can decide whether to retry, report, or suppress it.

## Retries and idempotency

Do not blindly retry every email error. Retry only transient failures, using the application's existing retry policy.

For operations that may be retried, use an idempotency key when supported by the installed SDK version. The key should be stable for the logical email operation, such as:

```text
welcome-email:<user-id>:<event-id>
```

Never generate a new idempotency key for every retry of the same logical operation.

## Testing

Prefer dependency injection around the email sender:

```go
type EmailSender interface {
	Send(ctx context.Context, message Message) (string, error)
}
```

Test:

- missing API key
- missing sender
- missing recipients
- missing subject
- missing content
- successful send
- Resend API failure
- context cancellation
- retry behavior, if implemented

Use a fake sender or `httptest.Server`. Tests must not send real emails.

## Implementation workflow

When asked to add Resend email functionality:

1. Inspect the repository's existing configuration and service patterns.
2. Check whether `github.com/resend/resend-go/v4` already exists in `go.mod`.
3. Add the dependency only if it is missing.
4. Create or extend an email service following local naming conventions.
5. Load `RESEND_API_KEY` securely.
6. Add context-aware sending and input validation.
7. Add unit tests with no real network calls.
8. Run `gofmt`.
9. Run the relevant tests and `go test ./...` when practical.
10. Summarize changed files, configuration requirements, and test results.
