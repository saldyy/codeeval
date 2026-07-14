// Package harness generates per-language starter code stubs and grading
// drivers from a problem's function signature. Every language ends up as a
// single concatenated source file: the user's submitted function definition
// followed by a small generated driver that reads JSON-encoded call
// arguments from stdin, calls the function, and prints the JSON-encoded
// return value to stdout.
//
// Supported types: int, float, string, bool, and one-dimensional arrays of
// each (int[], float[], string[], bool[]). No nested arrays, no objects, no
// multiple return values.
package harness

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Param struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Signature struct {
	FunctionName string  `json:"function_name"`
	Params       []Param `json:"params"`
	ReturnType   string  `json:"return_type"`
}

// ParseSignature unmarshals a problem's function_signature JSONB column.
func ParseSignature(raw []byte) (Signature, error) {
	var sig Signature
	if err := json.Unmarshal(raw, &sig); err != nil {
		return Signature{}, fmt.Errorf("parsing function signature: %w", err)
	}
	return sig, nil
}

func isArray(t string) bool    { return strings.HasSuffix(t, "[]") }
func elemType(t string) string { return strings.TrimSuffix(t, "[]") }

// Stub returns the starter code shown in the editor for the given language.
func Stub(sig Signature, language string) string {
	switch language {
	case "javascript":
		return stubJS(sig)
	case "python3":
		return stubPython(sig)
	case "go":
		return stubGo(sig)
	case "java":
		return stubJava(sig)
	case "c":
		return stubC(sig)
	case "cpp":
		return stubCpp(sig)
	default:
		return ""
	}
}

// Wrap concatenates the user's submitted source with a generated driver
// that reads the JSON-encoded call arguments from stdin, calls the user's
// function, and prints the JSON-encoded return value to stdout.
func Wrap(sig Signature, language, userSource string) string {
	switch language {
	case "javascript":
		return wrapJS(sig, userSource)
	case "python3":
		return wrapPython(sig, userSource)
	case "go":
		return wrapGo(sig, userSource)
	case "java":
		return wrapJava(sig, userSource)
	case "c":
		return wrapC(sig, userSource)
	case "cpp":
		return wrapCpp(sig, userSource)
	default:
		return userSource
	}
}
