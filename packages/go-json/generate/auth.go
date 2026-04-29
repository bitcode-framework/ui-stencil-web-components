package generate

import (
	"encoding/json"
)

// GenerateAuth generates a go-json auth scaffold with register, login, refresh, me, change-password.
func GenerateAuth() (string, error) {
	program := map[string]any{
		"name":    "auth-api",
		"go_json": "1",
		"server": map[string]any{
			"port": 3000,
			"jwt": map[string]any{
				"secret_env": "JWT_SECRET",
				"algorithm":  "HS256",
				"expiry":     "24h",
			},
		},
		"import": map[string]any{
			"db": "io:sql",
		},
		"middleware": []string{"logger", "recover", "cors"},
		"routes": []map[string]any{
			{"method": "POST", "path": "/auth/register", "handler": "register"},
			{"method": "POST", "path": "/auth/login", "handler": "login"},
			{"method": "POST", "path": "/auth/refresh", "handler": "refreshToken", "middleware": []string{"jwt"}},
			{"method": "GET", "path": "/auth/me", "handler": "getProfile", "middleware": []string{"jwt"}},
			{"method": "PUT", "path": "/auth/password", "handler": "changePassword", "middleware": []string{"jwt"}},
		},
		"functions": map[string]any{
			"register": map[string]any{
				"steps": []map[string]any{
					{"let": "email", "expr": "request.body.email"},
					{"let": "password", "expr": "request.body.password"},
					{"if": "email == '' || password == ''", "then": []map[string]any{
						{"return": map[string]any{"status": 400, "body": map[string]string{"error": "'Email and password are required'"}}},
					}},
					{"let": "hashed", "call": "crypto.sha256", "with": map[string]string{"value": "password"}},
					{"let": "result", "call": "db.execute", "with": map[string]string{
						"query": "INSERT INTO users (email, password_hash) VALUES (?, ?)",
						"args":  "[email, hashed]",
					}},
					{"return": map[string]any{"status": 201, "body": map[string]string{"id": "result.last_insert_id"}}},
				},
			},
			"login": map[string]any{
				"steps": []map[string]any{
					{"let": "email", "expr": "request.body.email"},
					{"let": "password", "expr": "request.body.password"},
					{"let": "users", "call": "db.query", "with": map[string]string{
						"query": "SELECT * FROM users WHERE email = ?",
						"args":  "[email]",
					}},
					{"if": "len(users.rows) == 0", "then": []map[string]any{
						{"return": map[string]any{"status": 401, "body": map[string]string{"error": "'Invalid credentials'"}}},
					}},
					{"let": "user", "expr": "users.rows[0]"},
					{"let": "hashed", "call": "crypto.sha256", "with": map[string]string{"value": "password"}},
					{"if": "hashed != user.password_hash", "then": []map[string]any{
						{"return": map[string]any{"status": 401, "body": map[string]string{"error": "'Invalid credentials'"}}},
					}},
					{"let": "token", "call": "jwt.sign", "with": map[string]string{
						"payload": "{sub: string(user.id), email: user.email}",
					}},
					{"return": map[string]any{"status": 200, "body": map[string]string{"token": "token", "user": "{id: user.id, email: user.email}"}}},
				},
			},
			"refreshToken": map[string]any{
				"steps": []map[string]any{
					{"let": "token", "call": "jwt.sign", "with": map[string]string{
						"payload": "{sub: request.user.sub, email: request.user.email}",
					}},
					{"return": map[string]any{"status": 200, "body": map[string]string{"token": "token"}}},
				},
			},
			"getProfile": map[string]any{
				"steps": []map[string]any{
					{"let": "users", "call": "db.query", "with": map[string]string{
						"query": "SELECT id, email, created_at FROM users WHERE id = ?",
						"args":  "[int(request.user.sub)]",
					}},
					{"if": "len(users.rows) == 0", "then": []map[string]any{
						{"return": map[string]any{"status": 404, "body": map[string]string{"error": "'User not found'"}}},
					}},
					{"return": map[string]any{"status": 200, "body": "users.rows[0]"}},
				},
			},
			"changePassword": map[string]any{
				"steps": []map[string]any{
					{"let": "newPassword", "expr": "request.body.new_password"},
					{"if": "newPassword == ''", "then": []map[string]any{
						{"return": map[string]any{"status": 400, "body": map[string]string{"error": "'New password is required'"}}},
					}},
					{"let": "hashed", "call": "crypto.sha256", "with": map[string]string{"value": "newPassword"}},
					{"let": "result", "call": "db.execute", "with": map[string]string{
						"query": "UPDATE users SET password_hash = ? WHERE id = ?",
						"args":  "[hashed, int(request.user.sub)]",
					}},
					{"return": map[string]any{"status": 200, "body": map[string]string{"message": "'Password updated'"}}},
				},
			},
		},
	}

	data, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GenerateAuthSQL returns the SQL to create the users table.
func GenerateAuthSQL() string {
	return `CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`
}
