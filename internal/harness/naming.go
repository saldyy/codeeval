package harness

import (
	"strings"
	"unicode"
)

// toSnakeCase converts a camelCase identifier (e.g. "fizzBuzz") to
// snake_case ("fizz_buzz") for Python stubs, the only language here that
// gets a naming-convention conversion from the canonical camelCase name.
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// toPascalCase converts a camelCase identifier (e.g. "fizzBuzz") to
// PascalCase ("FizzBuzz") for exported Go function names.
func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
