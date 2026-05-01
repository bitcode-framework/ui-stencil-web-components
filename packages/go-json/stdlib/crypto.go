package stdlib

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CryptoNamespace returns a map of crypto functions for injection as "crypto" variable.
func CryptoNamespace() map[string]any {
	return map[string]any{
		"sha256": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.sha256: requires a string argument")
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.sha256: argument must be a string")
			}
			h := sha256.Sum256([]byte(s))
			return hex.EncodeToString(h[:]), nil
		},
		"md5": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.md5: requires a string argument")
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.md5: argument must be a string")
			}
			h := md5.Sum([]byte(s))
			return hex.EncodeToString(h[:]), nil
		},
		"uuid": func(args ...any) (any, error) {
			return uuid.New().String(), nil
		},
		"hmac": func(args ...any) (any, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("crypto.hmac: requires (data, key) arguments")
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.hmac: first argument must be a string")
			}
			key, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.hmac: second argument must be a key string")
			}
			algo := "sha256"
			if len(args) > 2 {
				if a, ok := args[2].(string); ok {
					algo = a
				}
			}
			var hashFunc func() hash.Hash
			switch algo {
			case "sha256":
				hashFunc = sha256.New
			case "sha512":
				hashFunc = sha512.New
			default:
				return nil, fmt.Errorf("crypto.hmac: unsupported algorithm '%s' (use sha256 or sha512)", algo)
			}
			mac := hmac.New(hashFunc, []byte(key))
			mac.Write([]byte(s))
			return hex.EncodeToString(mac.Sum(nil)), nil
		},
		"sha512": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.sha512: requires a string argument")
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.sha512: argument must be a string")
			}
			h := sha512.Sum512([]byte(s))
			return hex.EncodeToString(h[:]), nil
		},
		"encrypt": func(args ...any) (any, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("crypto.encrypt: requires (plaintext, key)")
			}
			plaintext, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.encrypt: first argument must be a string")
			}
			key, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.encrypt: second argument must be a string key")
			}
			return aesGCMEncrypt(plaintext, key)
		},
		"decrypt": func(args ...any) (any, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("crypto.decrypt: requires (ciphertext, key)")
			}
			ciphertext, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.decrypt: first argument must be a string")
			}
			key, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.decrypt: second argument must be a string key")
			}
			return aesGCMDecrypt(ciphertext, key)
		},
		"hashPassword": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.hashPassword: requires a string argument")
			}
			password, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.hashPassword: argument must be a string")
			}
			hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
			if err != nil {
				return nil, fmt.Errorf("crypto.hashPassword: %s", err)
			}
			return string(hashed), nil
		},
		"verifyPassword": func(args ...any) (any, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("crypto.verifyPassword: requires (password, hash)")
			}
			password, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.verifyPassword: first argument must be a string")
			}
			hashStr, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.verifyPassword: second argument must be a string")
			}
			err := bcrypt.CompareHashAndPassword([]byte(hashStr), []byte(password))
			return err == nil, nil
		},
		"randomBytes": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.randomBytes: requires a number argument")
			}
			nF, ok := toFloat64(args[0])
			if !ok {
				return nil, fmt.Errorf("crypto.randomBytes: argument must be a number")
			}
			n := int(nF)
			if n <= 0 || n > 1024 {
				return nil, fmt.Errorf("crypto.randomBytes: size must be 1-1024")
			}
			b := make([]byte, n)
			if _, err := cryptoRand.Read(b); err != nil {
				return nil, fmt.Errorf("crypto.randomBytes: %s", err)
			}
			return hex.EncodeToString(b), nil
		},
	}
}

func aesGCMEncrypt(plaintext, key string) (string, error) {
	keyBytes := deriveAESKey(key)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("crypto.encrypt: %s", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto.encrypt: %s", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := cryptoRand.Read(nonce); err != nil {
		return "", fmt.Errorf("crypto.encrypt: %s", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func aesGCMDecrypt(ciphertextB64, key string) (string, error) {
	keyBytes := deriveAESKey(key)
	data, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("crypto.decrypt: invalid base64: %s", err)
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("crypto.decrypt: %s", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto.decrypt: %s", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("crypto.decrypt: ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto.decrypt: %s", err)
	}
	return string(plaintext), nil
}

// deriveAESKey pads or truncates key to exactly 32 bytes (AES-256).
func deriveAESKey(key string) []byte {
	h := sha256.Sum256([]byte(key))
	return h[:]
}
