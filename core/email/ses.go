package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

// EmailSender defines the interface for sending emails.
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// SESEmailSender implements EmailSender using AWS SES.
type SESEmailSender struct {
	sesClient *ses.Client
	from      string // Sender email address
}

var emailProviderRegistry = make(map[string]func(settings map[string]string) (EmailSender, error))

// RegisterEmailProvider allows registration of a new email provider by name.
func RegisterEmailProvider(name string, factory func(settings map[string]string) (EmailSender, error)) {
	emailProviderRegistry[name] = factory
}

// GetEmailSenderFromSettings returns an EmailSender based on settings (provider key required, defaults to ses).
func GetEmailSenderFromSettings(settings map[string]string) (EmailSender, error) {
	provider := settings["provider"]
	if provider == "" {
		provider = "ses"
	}
	factory, ok := emailProviderRegistry[provider]
	if !ok {
		return nil, fmt.Errorf("email provider '%s' not registered", provider)
	}
	return factory(settings)
}

// SES provider factory for the registry.
func sesProviderFactory(settings map[string]string) (EmailSender, error) {
	from := settings["from_address"]
	region := settings["aws_region"]
	accessKey := settings["aws_access_key_id"]
	secretKey := settings["aws_secret_access_key"]
	return NewSESEmailSender(context.Background(), from, region, accessKey, secretKey)
}

func init() {
	RegisterEmailProvider("ses", sesProviderFactory)
}

// NewSESEmailSender creates a new SES email sender.
// If accessKey and secretKey are provided, use them; otherwise, use default credentials.
func NewSESEmailSender(ctx context.Context, from, awsRegion, accessKey, secretKey string) (*SESEmailSender, error) {
	var cfg aws.Config
	var err error
	if accessKey != "" && secretKey != "" {
		creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsRegion),
			config.WithCredentialsProvider(creds),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(awsRegion))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := ses.NewFromConfig(cfg)
	return &SESEmailSender{sesClient: client, from: from}, nil
}

// SendEmail sends an email using AWS SES.
func (s *SESEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	input := &ses.SendEmailInput{
		Source: aws.String(s.from),
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Message: &types.Message{
			Subject: &types.Content{Data: aws.String(subject)},
			Body: &types.Body{
				Text: &types.Content{Data: aws.String(body)},
			},
		},
	}
	_, err := s.sesClient.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

// TODO: Add configuration for credentials, error handling, and support for HTML emails if needed.
// TODO: Add provider abstraction for EmailSender to allow swapping providers in the future.
