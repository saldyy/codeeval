package harness

import (
	"fmt"
	"strings"
)

func jsType(t string) string {
	if isArray(t) {
		return jsType(elemType(t)) + "[]"
	}
	switch t {
	case "int", "float":
		return "number"
	case "bool":
		return "boolean"
	default:
		return t // "string"
	}
}

func stubJS(sig Signature) string {
	var doc, params []string
	for _, p := range sig.Params {
		doc = append(doc, fmt.Sprintf(" * @param {%s} %s", jsType(p.Type), p.Name))
		params = append(params, p.Name)
	}
	doc = append(doc, fmt.Sprintf(" * @return {%s}", jsType(sig.ReturnType)))
	return fmt.Sprintf("/**\n%s\n */\nvar %s = function(%s) {\n    \n};\n",
		strings.Join(doc, "\n"), sig.FunctionName, strings.Join(params, ", "))
}

func wrapJS(sig Signature, userSource string) string {
	return fmt.Sprintf(`%s

const __args = JSON.parse(require('fs').readFileSync(0, 'utf8'));
const __result = %s(...__args);
console.log(JSON.stringify(__result));
`, userSource, sig.FunctionName)
}
