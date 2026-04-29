package server

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/server/adapters"
	"github.com/golang-jwt/jwt/v5"
)

// JWTMiddleware creates a middleware that validates JWT tokens.
func JWTMiddleware(cfg *lang.JWTConfig) adapters.MiddlewareFunc {
	return func(ctx *adapters.RequestContext, next func() *adapters.Response) *adapters.Response {
		tokenStr := extractToken(ctx, cfg)
		if tokenStr == "" {
			return unauthorizedResponse("missing token")
		}

		secret := os.Getenv(cfg.SecretEnv)
		if secret == "" {
			return &adapters.Response{
				Status: 500,
				Body: map[string]any{
					"error": map[string]any{
						"code":    "JWT_CONFIG_ERROR",
						"message": fmt.Sprintf("environment variable %q not set", cfg.SecretEnv),
					},
				},
			}
		}

		claims, err := verifyToken(tokenStr, secret, cfg.Algorithm)
		if err != nil {
			return unauthorizedResponse(err.Error())
		}

		ctx.User = claims
		return next()
	}
}

// JWTFunctions returns callable JWT functions for use in go-json handlers.
func JWTFunctions(cfg *lang.JWTConfig) map[string]any {
	secret := os.Getenv(cfg.SecretEnv)
	algorithm := resolveAlgorithm(cfg.Algorithm)

	return map[string]any{
		"sign": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("jwt.sign requires at least 1 argument (payload)")
			}
			payload, ok := args[0].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("jwt.sign: payload must be a map")
			}

			expiry := cfg.Expiry
			if len(args) > 1 {
				if e, ok := args[1].(string); ok {
					expiry = e
				}
			}

			return signToken(payload, secret, algorithm, expiry)
		},
		"verify": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("jwt.verify requires 1 argument (token)")
			}
			tokenStr, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("jwt.verify: token must be a string")
			}
			return verifyToken(tokenStr, secret, cfg.Algorithm)
		},
		"decode": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("jwt.decode requires 1 argument (token)")
			}
			tokenStr, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("jwt.decode: token must be a string")
			}
			return decodeToken(tokenStr)
		},
		"refresh": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("jwt.refresh requires at least 1 argument (token)")
			}
			tokenStr, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("jwt.refresh: token must be a string")
			}

			expiry := cfg.Expiry
			if len(args) > 1 {
				if e, ok := args[1].(string); ok {
					expiry = e
				}
			}

			claims, err := verifyToken(tokenStr, secret, cfg.Algorithm)
			if err != nil {
				return nil, fmt.Errorf("jwt.refresh: %w", err)
			}

			claimsMap, ok := claims.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("jwt.refresh: invalid claims")
			}

			delete(claimsMap, "exp")
			delete(claimsMap, "iat")
			delete(claimsMap, "nbf")

			return signToken(claimsMap, secret, algorithm, expiry)
		},
	}
}

func extractToken(ctx *adapters.RequestContext, cfg *lang.JWTConfig) string {
	header := cfg.Header
	if header == "" {
		header = "Authorization"
	}
	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "Bearer "
	}

	if authHeader, ok := ctx.Headers[header]; ok {
		if strings.HasPrefix(authHeader, prefix) {
			return strings.TrimPrefix(authHeader, prefix)
		}
	}

	if cfg.Cookie != "" {
		if token, ok := ctx.Cookies[cfg.Cookie]; ok && token != "" {
			return token
		}
	}

	return ""
}

func signToken(payload map[string]any, secret, algorithm, expiry string) (string, error) {
	claims := jwt.MapClaims{}
	for k, v := range payload {
		claims[k] = v
	}

	claims["iat"] = time.Now().Unix()

	if expiry != "" {
		dur, err := time.ParseDuration(expiry)
		if err == nil {
			claims["exp"] = time.Now().Add(dur).Unix()
		}
	}

	method := getSigningMethod(algorithm)
	token := jwt.NewWithClaims(method, claims)
	return token.SignedString([]byte(secret))
}

func verifyToken(tokenStr, secret, algorithm string) (any, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		expected := getSigningMethod(algorithm)
		if token.Method.Alg() != expected.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		result := make(map[string]any)
		for k, v := range claims {
			result[k] = v
		}
		return result, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

func decodeToken(tokenStr string) (any, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("cannot decode token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		result := make(map[string]any)
		for k, v := range claims {
			result[k] = v
		}
		return result, nil
	}

	return nil, fmt.Errorf("cannot decode token claims")
}

func resolveAlgorithm(alg string) string {
	if alg == "" {
		return "HS256"
	}
	return alg
}

func getSigningMethod(algorithm string) jwt.SigningMethod {
	switch strings.ToUpper(algorithm) {
	case "HS384":
		return jwt.SigningMethodHS384
	case "HS512":
		return jwt.SigningMethodHS512
	default:
		return jwt.SigningMethodHS256
	}
}

func unauthorizedResponse(message string) *adapters.Response {
	return &adapters.Response{
		Status: 401,
		Body: map[string]any{
			"error": map[string]any{
				"code":    "UNAUTHORIZED",
				"message": message,
			},
		},
	}
}
