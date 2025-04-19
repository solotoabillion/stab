package email

import (
	"context"
	"testing"
)

func TestSESEmailSender_InvalidCredentials(t *testing.T) {
	sender, err := NewSESEmailSender(context.Background(), "test@example.com", "us-east-1", "fake-access", "fake-secret")
	if err != nil {
		t.Fatalf("unexpected error creating SES sender: %v", err)
	}
	err = sender.SendEmail(context.Background(), "to@example.com", "Test Subject", "Test Body")
	if err == nil {
		t.Error("expected error sending email with fake credentials, got nil")
	}
}

func TestProviderRegistryAndFactory(t *testing.T) {
	settings := map[string]string{
		"from_address":          "test@example.com",
		"aws_region":            "us-east-1",
		"aws_access_key_id":     "fake-access",
		"aws_secret_access_key": "fake-secret",
		"provider":              "ses",
	}
	sender, err := GetEmailSenderFromSettings(settings)
	if err != nil {
		t.Fatalf("unexpected error from provider factory: %v", err)
	}
	if sender == nil {
		t.Error("expected non-nil sender from provider factory")
	}

	settings["provider"] = "notarealprovider"
	_, err = GetEmailSenderFromSettings(settings)
	if err == nil {
		t.Error("expected error for unknown provider, got nil")
	}
}
