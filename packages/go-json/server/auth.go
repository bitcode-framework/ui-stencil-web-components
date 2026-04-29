package server

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/runtime"
	"github.com/bitcode-framework/go-json/server/adapters"
)

// AuthStrategy authenticates a request and returns a user object or error.
type AuthStrategy interface {
	Name() string
	Type() string
	Authenticate(ctx *adapters.RequestContext) (any, error)
}

// AuthRegistry holds registered auth strategies and resolves middleware names.
type AuthRegistry struct {
	strategies      map[string]AuthStrategy
	defaultStrategy string
}

// NewAuthRegistry creates an AuthRegistry from server config.
func NewAuthRegistry(cfg *lang.AuthConfig, jwtCfg *lang.JWTConfig, rt *runtime.Runtime, compiled *lang.CompiledProgram) *AuthRegistry {
	ar := &AuthRegistry{
		strategies: make(map[string]AuthStrategy),
	}

	if cfg == nil {
		if jwtCfg != nil {
			ar.strategies["jwt"] = &BearerStrategy{config: jwtCfg}
			ar.defaultStrategy = "jwt"
		}
		return ar
	}

	ar.defaultStrategy = cfg.Default

	for name, sc := range cfg.Strategies {
		switch sc.Type {
		case "bearer":
			jc := &lang.JWTConfig{
				SecretEnv: sc.SecretEnv,
				Algorithm: sc.Algorithm,
				Expiry:    sc.Expiry,
				Cookie:    sc.Cookie,
				Header:    sc.Header,
				Prefix:    sc.Prefix,
				Claims:    sc.Claims,
			}
			ar.strategies[name] = &BearerStrategy{name: name, config: jc}
		case "apikey":
			ar.strategies[name] = &APIKeyStrategy{
				name:       name,
				header:     sc.Header,
				queryParam: sc.QueryParam,
				keysEnv:    sc.KeysEnv,
			}
		case "basic":
			ar.strategies[name] = &BasicStrategy{
				name:     name,
				usersEnv: sc.UsersEnv,
				realm:    sc.Realm,
			}
		case "custom":
			ar.strategies[name] = &CustomAuthStrategy{
				name:     name,
				handler:  sc.Handler,
				rt:       rt,
				compiled: compiled,
			}
		}
	}

	return ar
}

// Middleware returns an adapters.MiddlewareFunc for the given auth spec.
// "auth" uses the default strategy, "auth:name" uses a specific one.
func (ar *AuthRegistry) Middleware(spec string) (adapters.MiddlewareFunc, error) {
	strategyName := ar.defaultStrategy
	if strings.HasPrefix(spec, "auth:") {
		strategyName = strings.TrimPrefix(spec, "auth:")
	} else if spec == "jwt" {
		strategyName = "jwt"
	} else if spec != "auth" {
		return nil, fmt.Errorf("unknown auth spec: %q", spec)
	}

	strategy, ok := ar.strategies[strategyName]
	if !ok {
		return nil, fmt.Errorf("auth strategy %q not found", strategyName)
	}

	return func(ctx *adapters.RequestContext, next func() *adapters.Response) *adapters.Response {
		user, err := strategy.Authenticate(ctx)
		if err != nil {
			return unauthorizedResponse(err.Error())
		}
		ctx.User = user
		return next()
	}, nil
}

// --- Bearer (JWT) Strategy ---

type BearerStrategy struct {
	name   string
	config *lang.JWTConfig
}

func (s *BearerStrategy) Name() string { return s.name }
func (s *BearerStrategy) Type() string { return "bearer" }

func (s *BearerStrategy) Authenticate(ctx *adapters.RequestContext) (any, error) {
	tokenStr := extractToken(ctx, s.config)
	if tokenStr == "" {
		return nil, fmt.Errorf("missing token")
	}

	secret := os.Getenv(s.config.SecretEnv)
	if secret == "" {
		return nil, fmt.Errorf("JWT secret not configured")
	}

	claims, err := verifyToken(tokenStr, secret, s.config.Algorithm)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

// --- API Key Strategy ---

type APIKeyStrategy struct {
	name       string
	header     string
	queryParam string
	keysEnv    string
}

func (s *APIKeyStrategy) Name() string { return s.name }
func (s *APIKeyStrategy) Type() string { return "apikey" }

func (s *APIKeyStrategy) Authenticate(ctx *adapters.RequestContext) (any, error) {
	var key string

	header := s.header
	if header == "" {
		header = "X-API-Key"
	}
	if k, ok := ctx.Headers[header]; ok && k != "" {
		key = k
	}

	if key == "" && s.queryParam != "" {
		if k, ok := ctx.Query[s.queryParam]; ok && k != "" {
			key = k
		}
	}

	if key == "" {
		return nil, fmt.Errorf("missing API key")
	}

	keysStr := os.Getenv(s.keysEnv)
	if keysStr == "" {
		return nil, fmt.Errorf("API keys not configured")
	}

	// Format: key1:name1,key2:name2
	for _, pair := range strings.Split(keysStr, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 2 && parts[0] == key {
			return map[string]any{
				"key":  key,
				"name": parts[1],
			}, nil
		}
		if len(parts) == 1 && parts[0] == key {
			return map[string]any{
				"key": key,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// --- Basic Auth Strategy ---

type BasicStrategy struct {
	name     string
	usersEnv string
	realm    string
}

func (s *BasicStrategy) Name() string { return s.name }
func (s *BasicStrategy) Type() string { return "basic" }

func (s *BasicStrategy) Authenticate(ctx *adapters.RequestContext) (any, error) {
	authHeader, ok := ctx.Headers["Authorization"]
	if !ok {
		authHeader, ok = ctx.Headers["authorization"]
	}
	if !ok || !strings.HasPrefix(authHeader, "Basic ") {
		return nil, fmt.Errorf("missing basic auth credentials")
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		return nil, fmt.Errorf("invalid basic auth encoding")
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid basic auth format")
	}
	username, password := parts[0], parts[1]

	usersStr := os.Getenv(s.usersEnv)
	if usersStr == "" {
		return nil, fmt.Errorf("basic auth users not configured")
	}

	// Format: user1:pass1,user2:pass2
	for _, pair := range strings.Split(usersStr, ",") {
		up := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(up) == 2 && up[0] == username && up[1] == password {
			return map[string]any{
				"username": username,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid credentials")
}

// --- Custom Auth Strategy ---

type CustomAuthStrategy struct {
	name     string
	handler  string
	rt       *runtime.Runtime
	compiled *lang.CompiledProgram
}

func (s *CustomAuthStrategy) Name() string { return s.name }
func (s *CustomAuthStrategy) Type() string { return "custom" }

func (s *CustomAuthStrategy) Authenticate(ctx *adapters.RequestContext) (any, error) {
	requestMap := BuildRequestMap(ctx)
	args := map[string]any{
		"request": requestMap,
	}

	result, err := s.rt.ExecuteFunction(s.compiled, s.handler, args)
	if err != nil {
		log.Printf("[go-json] custom auth %q error: %v", s.handler, err)
		return nil, fmt.Errorf("auth error: %v", err)
	}

	if result == nil {
		return nil, fmt.Errorf("auth handler returned nil")
	}

	if resultMap, ok := result.(map[string]any); ok {
		if _, hasStatus := resultMap["status"]; hasStatus {
			return nil, fmt.Errorf("authentication failed")
		}
	}

	return result, nil
}
