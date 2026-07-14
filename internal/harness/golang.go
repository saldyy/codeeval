package harness

import (
	"fmt"
	"strings"
)

func goType(t string) string {
	if isArray(t) {
		return "[]" + goType(elemType(t))
	}
	switch t {
	case "int":
		return "int"
	case "float":
		return "float64"
	case "bool":
		return "bool"
	default:
		return "string"
	}
}

// stubGo emits `package main` since wrapGo appends more declarations
// (imports, func main) to the same file - Go allows multiple import blocks
// and top-level funcs per file as long as `package` appears once.
func stubGo(sig Signature) string {
	name := toPascalCase(sig.FunctionName)
	var params []string
	for _, p := range sig.Params {
		params = append(params, fmt.Sprintf("%s %s", p.Name, goType(p.Type)))
	}
	return fmt.Sprintf("package main\n\nfunc %s(%s) %s {\n\t\n}\n",
		name, strings.Join(params, ", "), goType(sig.ReturnType))
}

// wrapGo must keep every import declaration ahead of every other top-level
// declaration in the file - the Go grammar requires ImportDecls before
// TopLevelDecls, so the harness's own import block can't simply be appended
// after the user's function. Instead: emit `package main` once, then the
// harness's imports, then the user's source with its own leading
// `package main` line stripped, then `func main`.
func wrapGo(sig Signature, userSource string) string {
	name := toPascalCase(sig.FunctionName)
	userBody := strings.TrimLeft(strings.TrimPrefix(strings.TrimSpace(userSource), "package main"), "\n")

	var driverBody strings.Builder
	var callArgs []string

	if len(sig.Params) > 0 {
		driverBody.WriteString("\t__data, _ := io.ReadAll(os.Stdin)\n")
		driverBody.WriteString("\tvar __raw []json.RawMessage\n")
		driverBody.WriteString("\tif err := json.Unmarshal(__data, &__raw); err != nil {\n\t\tpanic(err)\n\t}\n")
		for i, p := range sig.Params {
			varName := fmt.Sprintf("__arg%d", i)
			driverBody.WriteString(fmt.Sprintf("\tvar %s %s\n", varName, goType(p.Type)))
			driverBody.WriteString(fmt.Sprintf("\tif err := json.Unmarshal(__raw[%d], &%s); err != nil {\n\t\tpanic(err)\n\t}\n", i, varName))
			callArgs = append(callArgs, varName)
		}
	}

	imports := []string{`"encoding/json"`, `"fmt"`}
	if len(sig.Params) > 0 {
		imports = append(imports, `"io"`, `"os"`)
	}

	return fmt.Sprintf(`package main

import (
	%s
)

%s

func main() {
%s	__result := %s(%s)
	__out, _ := json.Marshal(__result)
	fmt.Println(string(__out))
}
`, strings.Join(imports, "\n\t"), userBody, driverBody.String(), name, strings.Join(callArgs, ", "))
}
