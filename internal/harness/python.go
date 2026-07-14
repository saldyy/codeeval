package harness

import (
	"fmt"
	"strings"
)

func stubPython(sig Signature) string {
	name := toSnakeCase(sig.FunctionName)
	var params []string
	for _, p := range sig.Params {
		params = append(params, p.Name)
	}
	return fmt.Sprintf("def %s(%s):\n    pass\n", name, strings.Join(params, ", "))
}

func wrapPython(sig Signature, userSource string) string {
	name := toSnakeCase(sig.FunctionName)
	return fmt.Sprintf(`%s

import json as __json
import sys as __sys
__args = __json.loads(__sys.stdin.read())
__result = %s(*__args)
print(__json.dumps(__result))
`, userSource, name)
}
