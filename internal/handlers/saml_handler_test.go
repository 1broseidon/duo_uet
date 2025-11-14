package handlers

import (
	"testing"
	"user_experience_toolkit/internal/config"
)

func TestExtractSAMLIntegrationKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "metadata URL",
			url:  "https://sso-example.sso.duosecurity.com/saml2/sp/DI123456789/metadata",
			want: "DI123456789",
		},
		{
			name: "sso URL",
			url:  "https://sso-example.sso.duosecurity.com/saml2/sp/DIABCDEFGHI/sso",
			want: "DIABCDEFGHI",
		},
		{
			name: "invalid URL",
			url:  "https://example.com/metadata",
			want: "",
		},
		{
			name: "empty input",
			url:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractSAMLIntegrationKey(tt.url); got != tt.want {
				t.Fatalf("extractSAMLIntegrationKey(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestSAMLHandlerResolveIntegrationKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		app  *config.Application
		want string
	}{
		{
			name: "client id preferred",
			app: &config.Application{
				ClientID: "DIEXISTINGKEY",
			},
			want: "DIEXISTINGKEY",
		},
		{
			name: "fallback to idp entity",
			app: &config.Application{
				IDPEntityID: "https://sso-example.sso.duosecurity.com/saml2/sp/DIFROMSAML/metadata",
			},
			want: "DIFROMSAML",
		},
		{
			name: "fallback to idp sso",
			app: &config.Application{
				IDPSSOURL: "https://sso-example.sso.duosecurity.com/saml2/sp/DIFFROMSSO/sso",
			},
			want: "DIFFROMSSO",
		},
		{
			name: "no data available",
			app:  &config.Application{},
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := &SAMLHandler{App: tt.app}
			if got := handler.resolveIntegrationKey(); got != tt.want {
				t.Fatalf("resolveIntegrationKey() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("nil handler safe", func(t *testing.T) {
		t.Parallel()
		var handler *SAMLHandler
		if got := handler.resolveIntegrationKey(); got != "" {
			t.Fatalf("nil handler resolveIntegrationKey() = %q, want empty string", got)
		}
	})
}
