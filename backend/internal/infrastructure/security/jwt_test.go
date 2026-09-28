package security

import (
	"testing"
	"time"
)

func TestJWTRoundTrip(t *testing.T) {
	service := NewJWT("a-secret-that-is-longer-than-thirty-two-characters", "test", time.Minute)
	token, err := service.Issue("user-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("UserID = %q", claims.UserID)
	}
}

func TestJWTRejectsInvalidToken(t *testing.T) {
	service := NewJWT("a-secret-that-is-longer-than-thirty-two-characters", "test", time.Minute)
	if _, err := service.Parse("not-a-token"); err == nil {
		t.Fatal("expected invalid token error")
	}
}
