package resend

import (
	"context"
	"fmt"
	"html"

	resendSDK "github.com/resend/resend-go/v4"
)

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

type EmailSender interface {
	Send(ctx context.Context, message Message) (string, error)
}

type Sender struct {
	client *resendSDK.Client
}

func NewSender(apiKey string) (*Sender, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("resend: API key is required")
	}

	return &Sender{
		client: resendSDK.NewClient(apiKey),
	}, nil
}

func (s *Sender) Send(ctx context.Context, message Message) (string, error) {
	if message.From == "" {
		return "", fmt.Errorf("resend.Send: sender address is required")
	}
	if len(message.To) == 0 {
		return "", fmt.Errorf("resend.Send: at least one recipient is required")
	}
	if message.Subject == "" {
		return "", fmt.Errorf("resend.Send: subject is required")
	}
	if message.HTML == "" && message.Text == "" {
		return "", fmt.Errorf("resend.Send: HTML or text content is required")
	}

	response, err := s.client.Emails.SendWithContext(ctx, &resendSDK.SendEmailRequest{
		From:    message.From,
		To:      message.To,
		Subject: message.Subject,
		Html:    message.HTML,
		Text:    message.Text,
		Cc:      message.Cc,
		Bcc:     message.Bcc,
		ReplyTo: message.ReplyTo,
	})
	if err != nil {
		return "", fmt.Errorf("resend.Send: send email: %w", err)
	}

	return response.Id, nil
}

func VerificationMessage(from, recipient, companyName string) Message {
	safeCompanyName := html.EscapeString(companyName)
	return Message{
		From:    from,
		To:      []string{recipient},
		Subject: "Bienvenue sur Nzo Business",
		HTML: fmt.Sprintf(
			"<p>Votre compte administrateur pour <strong>%s</strong> a été créé.</p><p>Votre adresse e-mail a été validée par notre service d'envoi.</p>",
			safeCompanyName,
		),
		Text: fmt.Sprintf(
			"Votre compte administrateur pour %s a été créé. Votre adresse e-mail a été validée par notre service d'envoi.",
			companyName,
		),
	}
}
