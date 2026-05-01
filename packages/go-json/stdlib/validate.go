package stdlib

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	emailRegex    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	uuidRegex     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	hexColorRegex = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
)

// ValidateNamespace returns a map of validation functions for injection as "validate" variable.
func ValidateNamespace() map[string]any {
	return map[string]any{
		"isEmail": func(args ...any) (any, error) {
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			return emailRegex.MatchString(s), nil
		},
		"isURL": func(args ...any) (any, error) {
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			u, err := url.Parse(s)
			return err == nil && u.Scheme != "" && u.Host != "", nil
		},
		"isIP": func(args ...any) (any, error) {
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			return net.ParseIP(s) != nil, nil
		},
		"isUUID": func(args ...any) (any, error) {
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			return uuidRegex.MatchString(strings.ToLower(s)), nil
		},
		"isJSON": func(args ...any) (any, error) {
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			return json.Valid([]byte(s)), nil
		},
		"isNumeric": func(args ...any) (any, error) {
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			_, err := strconv.ParseFloat(s, 64)
			return err == nil, nil
		},
		"isAlpha": func(args ...any) (any, error) {
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			if s == "" {
				return false, nil
			}
			for _, r := range s {
				if !unicode.IsLetter(r) {
					return false, nil
				}
			}
			return true, nil
		},
		"isBase64": func(args ...any) (any, error) {
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			if s == "" {
				return false, nil
			}
			_, err := base64.StdEncoding.DecodeString(s)
			return err == nil, nil
		},
		"isHexColor": func(args ...any) (any, error) {
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			return hexColorRegex.MatchString(s), nil
		},
		"isCreditCard": func(args ...any) (any, error) {
			s, ok := args[0].(string)
			if !ok {
				return false, nil
			}
			return luhnCheck(s), nil
		},
	}
}

func luhnCheck(number string) bool {
	digits := strings.ReplaceAll(number, " ", "")
	digits = strings.ReplaceAll(digits, "-", "")
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	sum := 0
	alt := false
	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')
		if n < 0 || n > 9 {
			return false
		}
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}
