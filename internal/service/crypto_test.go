package service_test

import (
	"testing"

	"subscription-tracker/internal/service"
)

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
		password  string
	}{
		{
			name:      "simple text",
			plaintext: "Hello, World!",
			password:  "testpassword123",
		},
		{
			name:      "JSON data",
			plaintext: `{"subscriptions":[{"name":"Netflix","amount":15.99}]}`,
			password:  "secure_password_456",
		},
		{
			name:      "empty string",
			plaintext: "",
			password:  "password",
		},
		{
			name:      "unicode text",
			plaintext: "Hello, 世界! 🌍",
			password:  "пароль123",
		},
		{
			name:      "long text",
			plaintext: string(make([]byte, 10000)), // 10KB of null bytes
			password:  "longpassword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := service.Encrypt([]byte(tt.plaintext), tt.password)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Encrypted should be different from plaintext
			if encrypted == tt.plaintext && tt.plaintext != "" {
				t.Error("encrypted text should not equal plaintext")
			}

			// Decrypt
			decrypted, err := service.Decrypt(encrypted, tt.password)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Decrypted should match original
			if string(decrypted) != tt.plaintext {
				t.Errorf("Decrypt() = %q, want %q", string(decrypted), tt.plaintext)
			}
		})
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	plaintext := "Secret message"
	password := "correct_password"
	wrongPassword := "wrong_password"

	// Encrypt
	encrypted, err := service.Encrypt([]byte(plaintext), password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Decrypt with wrong password should fail
	_, err = service.Decrypt(encrypted, wrongPassword)
	if err == nil {
		t.Error("Decrypt() with wrong password should fail")
	}
}

func TestDecryptInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"empty string", ""},
		{"invalid base64", "not-valid-base64!!!"},
		{"too short", "YWJj"}, // "abc" in base64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Decrypt(tt.data, "password")
			if err == nil {
				t.Error("Decrypt() with invalid data should fail")
			}
		})
	}
}

func TestEncryptProducesDifferentOutputs(t *testing.T) {
	plaintext := "Same message"
	password := "same_password"

	// Encrypt twice
	encrypted1, err := service.Encrypt([]byte(plaintext), password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	encrypted2, err := service.Encrypt([]byte(plaintext), password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Due to random salt and nonce, outputs should be different
	if encrypted1 == encrypted2 {
		t.Error("Encrypt() should produce different outputs for same input (random salt/nonce)")
	}

	// But both should decrypt to the same plaintext
	decrypted1, _ := service.Decrypt(encrypted1, password)
	decrypted2, _ := service.Decrypt(encrypted2, password)

	if string(decrypted1) != string(decrypted2) {
		t.Error("Both encrypted versions should decrypt to the same plaintext")
	}
}
