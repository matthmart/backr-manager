package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (srv *server) authenticateRequest(ctx context.Context) error {

	// fetch all user accounts
	accounts, err := srv.AccountRepo.List()
	if err != nil {
		log.Info().Err(err).Msg("unable to check for accounts count")
	} else if len(accounts) == 0 {
		// if no account is available, consider authentication is successful
		log.Warn().Msg("API is not secured: an account must be created")
		return nil
	}

	// extract Authorization header
	auth, err := extractHeader(ctx, "authorization")
	if err != nil {
		return status.Error(codes.Unauthenticated, `missing "Authorization" header`)
	}

	// check for Bearer prefix
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return status.Error(codes.Unauthenticated, `missing "Bearer " prefix in "Authorization" header`)
	}

	// extract the token
	token := strings.TrimPrefix(auth, prefix)

	// parse the JWT token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// validate the alg
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(srv.Config.JWTSecret), nil
	})
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "unable to parse token: %v", err)
	}

	// check if the token is valid
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
