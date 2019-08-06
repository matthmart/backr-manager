package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/agence-webup/backr/manager"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func authenticateRequest(ctx context.Context, authConfig manager.AuthConfig) error {
	auth, err := extractHeader(ctx, "authorization")
	if err != nil {
		return status.Error(codes.Unauthenticated, `missing "Authorization" header`)
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return status.Error(codes.Unauthenticated, `missing "Bearer " prefix in "Authorization" header`)
	}

	token := strings.TrimPrefix(auth, prefix)

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// validate the alg
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(authConfig.JWTSecret), nil
	})
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "unable to parse token: %v", err)
	}

	if _, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		return nil
	}

	return status.Error(codes.Unauthenticated, "invalid token")
}

func extractHeader(ctx context.Context, header string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no headers in request")
	}

	authHeaders, ok := md[header]
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no header in request")
	}

	if len(authHeaders) != 1 {
		return "", status.Error(codes.Unauthenticated, "more than 1 header in request")
	}

	return authHeaders[0], nil
}
