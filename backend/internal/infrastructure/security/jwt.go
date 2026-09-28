package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/meta-super-app/backend/internal/domain"
)

type JWT struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

func NewJWT(secret, issuer string, ttl time.Duration) JWT {
	return JWT{secret: []byte(secret), issuer: issuer, ttl: ttl}
}
func (j JWT) Issue(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{Subject: userID, Issuer: j.issuer, ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl))}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
}
func (j JWT) Parse(value string) (*domain.TokenClaims, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(value, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	}, jwt.WithIssuer(j.issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.Subject == "" {
		return nil, errors.New("invalid token")
	}
	return &domain.TokenClaims{UserID: claims.Subject}, nil
}
