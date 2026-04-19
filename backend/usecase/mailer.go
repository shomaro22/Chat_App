package usecase

import (
	"context"
	"log"
)

// Mailer は認証メール等の送信抽象（SMTP 未設定時は LogMailer）。
type Mailer interface {
	Send(ctx context.Context, toEmail, subject, bodyText string) error
}

// LogMailer は本文をログに出すだけの開発用実装。
type LogMailer struct{}

func (LogMailer) Send(ctx context.Context, toEmail, subject, bodyText string) error {
	log.Printf("mail to=%s subject=%s\n%s", toEmail, subject, bodyText)
	return nil
}
