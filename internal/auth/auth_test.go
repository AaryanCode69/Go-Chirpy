package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	secret := "my-secret"

	tokenString, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}
	if tokenString == "" {
		t.Fatal("MakeJWT returned an empty token string")
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	correctSecret := "correct-secret"
	wrongSecret := "wrong-secret"

	validToken, err := MakeJWT(userID, correctSecret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	expiredToken, err := MakeJWT(userID, correctSecret, -time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	tests := []struct {
		name        string
		tokenString string
		secret      string
		wantUserID  uuid.UUID
		wantErr     bool
	}{
		{
			name:        "valid token",
			tokenString: validToken,
			secret:      correctSecret,
			wantUserID:  userID,
			wantErr:     false,
		},
		{
			name:        "expired token is rejected",
			tokenString: expiredToken,
			secret:      correctSecret,
			wantErr:     true,
		},
		{
			name:        "wrong secret is rejected",
			tokenString: validToken,
			secret:      wrongSecret,
			wantErr:     true,
		},
		{
			name:        "malformed token is rejected",
			tokenString: "not.a.valid.jwt",
			secret:      correctSecret,
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotID, err := ValidateJWT(tc.tokenString, tc.secret)

			if tc.wantErr {
				if err == nil {
					t.Error("expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if gotID != tc.wantUserID {
				t.Errorf("expected userID %v, got %v", tc.wantUserID, gotID)
			}
		})
	}
}
