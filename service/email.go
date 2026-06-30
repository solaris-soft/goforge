package service

import (
	"context"

	"github.com/resend/resend-go/v2"
)

type EmailClient struct {
	client *resend.Client
	from   string
}

func NewEmailClient(apiKey, fromAddr string) *EmailClient {
	return &EmailClient{
		client: resend.NewClient(apiKey),
		from:   fromAddr,
	}
}

func (e *EmailClient) Send(
	ctx context.Context,
	to []string,
	subject string,
	html string,
) error {
	params := &resend.SendEmailRequest{
		From:    e.from,
		To:      to,
		Subject: subject,
		Html:    html,
	}

	_, err := e.client.Emails.SendWithContext(ctx, params)
	return err
}
