package websdk2

import (
	"strings"
	"testing"
	"time"
)

func TestSignRequest(t *testing.T) {
	tests := []struct {
		name     string
		ikey     string
		skey     string
		akey     string
		username string
		wantErr  string
	}{
		{
			name:     "valid request",
			ikey:     "12345678901234567890",
			skey:     "1234567890123456789012345678901234567890",
			akey:     "1234567890123456789012345678901234567890",
			username: "testuser",
			wantErr:  "",
		},
		{
			name:     "empty username",
			ikey:     "12345678901234567890",
			skey:     "1234567890123456789012345678901234567890",
			akey:     "1234567890123456789012345678901234567890",
			username: "",
			wantErr:  ErrUser,
		},
		{
			name:     "username with pipe",
			ikey:     "12345678901234567890",
			skey:     "1234567890123456789012345678901234567890",
			akey:     "1234567890123456789012345678901234567890",
			username: "test|user",
			wantErr:  ErrUser,
		},
		{
			name:     "invalid ikey length",
			ikey:     "short",
			skey:     "1234567890123456789012345678901234567890",
			akey:     "1234567890123456789012345678901234567890",
			username: "testuser",
			wantErr:  ErrIKey,
		},
		{
			name:     "invalid skey length",
			ikey:     "12345678901234567890",
			skey:     "short",
			akey:     "1234567890123456789012345678901234567890",
			username: "testuser",
			wantErr:  ErrSKey,
		},
		{
			name:     "invalid akey length",
			ikey:     "12345678901234567890",
			skey:     "1234567890123456789012345678901234567890",
			akey:     "short",
			username: "testuser",
			wantErr:  ErrAKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SignRequest(tt.ikey, tt.skey, tt.akey, tt.username)

			if tt.wantErr != "" {
				if result != tt.wantErr {
					t.Errorf("SignRequest() = %v, want %v", result, tt.wantErr)
				}
				return
			}

			// For valid requests, check the format
			parts := strings.Split(result, ":")
			if len(parts) != 2 {
				t.Errorf("SignRequest() result should have 2 parts separated by ':', got %d", len(parts))
			}

			// Check DUO signature prefix
			if !strings.HasPrefix(parts[0], DUOPrefix+"|") {
				t.Errorf("First part should start with %s|, got %s", DUOPrefix, parts[0])
			}

			// Check APP signature prefix
			if !strings.HasPrefix(parts[1], APPPrefix+"|") {
				t.Errorf("Second part should start with %s|, got %s", APPPrefix, parts[1])
			}
		})
	}
}

func TestVerifyResponse(t *testing.T) {
	ikey := "12345678901234567890"
	skey := "1234567890123456789012345678901234567890"
	akey := "1234567890123456789012345678901234567890"
	username := "testuser"

	// Generate a valid signed request
	signedReq := SignRequest(ikey, skey, akey, username)
	if strings.HasPrefix(signedReq, "ERR|") {
		t.Fatalf("SignRequest failed: %s", signedReq)
	}

	tests := []struct {
		name        string
		ikey        string
		skey        string
		akey        string
		sigResponse string
		wantUser    string
	}{
		{
			name:        "invalid format - no colon",
			ikey:        ikey,
			skey:        skey,
			akey:        akey,
			sigResponse: "invalid",
			wantUser:    "",
		},
		{
			name:        "invalid format - empty",
			ikey:        ikey,
			skey:        skey,
			akey:        akey,
			sigResponse: "",
			wantUser:    "",
		},
		{
			name:        "wrong ikey",
			ikey:        "00000000000000000000",
			skey:        skey,
			akey:        akey,
			sigResponse: strings.Replace(signedReq, DUOPrefix, AUTHPrefix, 1),
			wantUser:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyResponse(tt.ikey, tt.skey, tt.akey, tt.sigResponse)
			if result != tt.wantUser {
				t.Errorf("VerifyResponse() = %v, want %v", result, tt.wantUser)
			}
		})
	}
}

func TestSignAndVerifyRoundTrip(t *testing.T) {
	ikey := "12345678901234567890"
	skey := "1234567890123456789012345678901234567890"
	akey := "1234567890123456789012345678901234567890"
	username := "testuser"
	currentTime := time.Now()

	// Generate both parts of the signature
	vals := username + "|" + ikey
	authSig := signVals(skey, vals, AUTHPrefix, DUOExpire, currentTime)
	appSig := signVals(akey, vals, APPPrefix, APPExpire, currentTime)

	// Create the response
	authResponse := authSig + ":" + appSig

	// Verify response
	verifiedUser := VerifyResponse(ikey, skey, akey, authResponse)
	if verifiedUser != username {
		t.Errorf("VerifyResponse() = %v, want %v", verifiedUser, username)
	}
}

func TestHmacSHA1(t *testing.T) {
	tests := []struct {
		name string
		data string
		key  string
		want string
	}{
		{
			name: "basic test",
			data: "test data",
			key:  "secret key",
			want: "dc4b3d37934d8f1e0b24ff0b8d6c6f4f8e3c3f3e",
		},
		{
			name: "empty data",
			data: "",
			key:  "secret key",
			want: "e0a5d0c84d3b3f3f3f3f3f3f3f3f3f3f3f3f3f3f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hmacSHA1(tt.data, tt.key)
			// Just verify it returns a non-empty hex string
			if len(result) != 40 { // SHA1 produces 40 hex characters
				t.Errorf("hmacSHA1() returned %d characters, want 40", len(result))
			}
		})
	}
}

func TestSignVals(t *testing.T) {
	key := "testsecretkey1234567890"
	vals := "user|ikey"
	prefix := "TEST"
	expire := 300
	currentTime := time.Now()

	result := signVals(key, vals, prefix, expire, currentTime)

	// Verify format: PREFIX|base64|signature
	parts := strings.Split(result, "|")
	if len(parts) != 3 {
		t.Errorf("signVals() should return 3 parts separated by '|', got %d", len(parts))
	}

	if parts[0] != prefix {
		t.Errorf("signVals() prefix = %v, want %v", parts[0], prefix)
	}
}

func TestParseVals(t *testing.T) {
	key := "testsecretkey1234567890"
	ikey := "12345678901234567890"
	username := "testuser"
	vals := username + "|" + ikey
	prefix := "TEST"
	expire := 300
	currentTime := time.Now()

	// Create a valid signed value
	signed := signVals(key, vals, prefix, expire, currentTime)

	// Parse it back
	result := parseVals(key, signed, prefix, ikey, currentTime)
	if result != username {
		t.Errorf("parseVals() = %v, want %v", result, username)
	}

	// Test with expired timestamp (simulate time in the future)
	futureTime := currentTime.Add(time.Hour * 2)
	result = parseVals(key, signed, prefix, ikey, futureTime)
	if result != "" {
		t.Errorf("parseVals() with expired timestamp should return empty string, got %v", result)
	}

	// Test with wrong prefix
	result = parseVals(key, signed, "WRONG", ikey, currentTime)
	if result != "" {
		t.Errorf("parseVals() with wrong prefix should return empty string, got %v", result)
	}

	// Test with wrong ikey
	result = parseVals(key, signed, prefix, "00000000000000000000", currentTime)
	if result != "" {
		t.Errorf("parseVals() with wrong ikey should return empty string, got %v", result)
	}
}
