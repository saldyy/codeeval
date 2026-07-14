package harness

import (
	"fmt"
	"strings"
)

// cHelpers is a fixed hand-rolled JSON parse/serialize library using
// bounded static buffers - C has no JSON, no std::string/vector, and no
// exceptions, so parsing is deliberately simple rather than fully general
// (fine at this app's small-test-case scale). Array params/returns follow
// LeetCode's own C convention: a pointer plus a separate `int` size
// parameter (params) or an `int* returnSize` out-parameter (return value),
// since plain C arrays don't carry their own length.
const cHelpers = `
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define __MAX_TOKENS 256
#define __MAX_TOKLEN 8192

static char __toks[__MAX_TOKENS][__MAX_TOKLEN];
static int __numToks;

static void __splitTop(const char* s) {
    int depth = 0, inStr = 0, pos = 0, any = 0;
    __numToks = 0;
    for (int i = 0; s[i]; i++) {
        char c = s[i];
        any = 1;
        if (inStr) {
            __toks[__numToks][pos++] = c;
            if (c == '\\' && s[i+1]) { __toks[__numToks][pos++] = s[++i]; continue; }
            if (c == '"') inStr = 0;
            continue;
        }
        if (c == '"') { inStr = 1; __toks[__numToks][pos++] = c; continue; }
        if (c == '[') depth++;
        if (c == ']') depth--;
        if (c == ',' && depth == 0) { __toks[__numToks][pos] = '\0'; __numToks++; pos = 0; continue; }
        __toks[__numToks][pos++] = c;
    }
    __toks[__numToks][pos] = '\0';
    if (pos > 0 || (any && __numToks == 0)) __numToks++;
}

static char* __trim(char* s) {
    while (*s == ' ' || *s == '\t' || *s == '\n' || *s == '\r') s++;
    int len = strlen(s);
    while (len > 0 && (s[len-1]==' '||s[len-1]=='\t'||s[len-1]=='\n'||s[len-1]=='\r')) { s[--len] = '\0'; }
    return s;
}

static char* __parseString(char* tok) {
    char* t = __trim(tok);
    int len = strlen(t);
    char* buf = (char*)malloc(len + 1);
    int bi = 0;
    for (int i = 1; i < len - 1; i++) {
        if (t[i] == '\\' && i + 1 < len - 1) { buf[bi++] = t[++i]; }
        else buf[bi++] = t[i];
    }
    buf[bi] = '\0';
    return buf;
}

static void __innerOf(char* tok, char* out) {
    char* t = __trim(tok);
    int len = strlen(t);
    if (len > 2) { strncpy(out, t + 1, len - 2); out[len - 2] = '\0'; } else { out[0] = '\0'; }
}

static int* __parseIntArray(char* tok, int* outCount) {
    char inner[__MAX_TOKLEN];
    __innerOf(tok, inner);
    __splitTop(inner);
    int* out = (int*)malloc(sizeof(int) * (__numToks > 0 ? __numToks : 1));
    for (int i = 0; i < __numToks; i++) out[i] = atoi(__trim(__toks[i]));
    *outCount = __numToks;
    return out;
}

static double* __parseDoubleArray(char* tok, int* outCount) {
    char inner[__MAX_TOKLEN];
    __innerOf(tok, inner);
    __splitTop(inner);
    double* out = (double*)malloc(sizeof(double) * (__numToks > 0 ? __numToks : 1));
    for (int i = 0; i < __numToks; i++) out[i] = atof(__trim(__toks[i]));
    *outCount = __numToks;
    return out;
}

static bool* __parseBoolArray(char* tok, int* outCount) {
    char inner[__MAX_TOKLEN];
    __innerOf(tok, inner);
    __splitTop(inner);
    bool* out = (bool*)malloc(sizeof(bool) * (__numToks > 0 ? __numToks : 1));
    for (int i = 0; i < __numToks; i++) out[i] = strcmp(__trim(__toks[i]), "true") == 0;
    *outCount = __numToks;
    return out;
}

static char** __parseStringArray(char* tok, int* outCount) {
    char inner[__MAX_TOKLEN];
    __innerOf(tok, inner);
    __splitTop(inner);
    char** out = (char**)malloc(sizeof(char*) * (__numToks > 0 ? __numToks : 1));
    for (int i = 0; i < __numToks; i++) out[i] = __parseString(__toks[i]);
    *outCount = __numToks;
    return out;
}

static void __printJSONString(const char* s) {
    putchar('"');
    for (int i = 0; s[i]; i++) {
        if (s[i] == '"' || s[i] == '\\') putchar('\\');
        putchar(s[i]);
    }
    putchar('"');
}
`

func cScalarType(t string) string {
	switch t {
	case "int":
		return "int"
	case "float":
		return "double"
	case "bool":
		return "bool"
	default:
		return "char*"
	}
}

// cParamList follows LeetCode's C convention: array params become a
// pointer plus a separate `<name>Size` int parameter.
func cParamList(sig Signature) []string {
	var parts []string
	for _, p := range sig.Params {
		if isArray(p.Type) {
			elem := elemType(p.Type)
			if elem == "string" {
				parts = append(parts, fmt.Sprintf("char** %s, int %sSize", p.Name, p.Name))
			} else {
				parts = append(parts, fmt.Sprintf("%s* %s, int %sSize", cScalarType(elem), p.Name, p.Name))
			}
		} else {
			parts = append(parts, fmt.Sprintf("%s %s", cScalarType(p.Type), p.Name))
		}
	}
	return parts
}

// cReturnType follows LeetCode's C convention: an array return value
// becomes a pointer return type plus a trailing `int* returnSize`
// out-parameter, since a bare C pointer carries no length.
func cReturnType(t string) (decl string, isArrayReturn bool) {
	if isArray(t) {
		elem := elemType(t)
		if elem == "string" {
			return "char**", true
		}
		return cScalarType(elem) + "*", true
	}
	return cScalarType(t), false
}

func stubC(sig Signature) string {
	retType, isArr := cReturnType(sig.ReturnType)
	params := cParamList(sig)
	if isArr {
		params = append(params, "int* returnSize")
	}
	return fmt.Sprintf("#include <stdio.h>\n#include <stdlib.h>\n#include <string.h>\n#include <stdbool.h>\n\n%s %s(%s) {\n    \n}\n",
		retType, sig.FunctionName, strings.Join(params, ", "))
}

func cParseParam(i int, p Param) (decl, callExpr string) {
	tokExpr := fmt.Sprintf("__args[%d]", i)
	switch p.Type {
	case "int":
		return fmt.Sprintf("    int %s = atoi(__trim(%s));", p.Name, tokExpr), p.Name
	case "float":
		return fmt.Sprintf("    double %s = atof(__trim(%s));", p.Name, tokExpr), p.Name
	case "bool":
		return fmt.Sprintf("    bool %s = strcmp(__trim(%s), \"true\") == 0;", p.Name, tokExpr), p.Name
	case "string":
		return fmt.Sprintf("    char* %s = __parseString(%s);", p.Name, tokExpr), p.Name
	case "int[]":
		return fmt.Sprintf("    int %sSize;\n    int* %s = __parseIntArray(%s, &%sSize);", p.Name, p.Name, tokExpr, p.Name),
			fmt.Sprintf("%s, %sSize", p.Name, p.Name)
	case "float[]":
		return fmt.Sprintf("    int %sSize;\n    double* %s = __parseDoubleArray(%s, &%sSize);", p.Name, p.Name, tokExpr, p.Name),
			fmt.Sprintf("%s, %sSize", p.Name, p.Name)
	case "bool[]":
		return fmt.Sprintf("    int %sSize;\n    bool* %s = __parseBoolArray(%s, &%sSize);", p.Name, p.Name, tokExpr, p.Name),
			fmt.Sprintf("%s, %sSize", p.Name, p.Name)
	case "string[]":
		return fmt.Sprintf("    int %sSize;\n    char** %s = __parseStringArray(%s, &%sSize);", p.Name, p.Name, tokExpr, p.Name),
			fmt.Sprintf("%s, %sSize", p.Name, p.Name)
	default:
		return "", p.Name
	}
}

func cSerializeReturn(t string) string {
	switch t {
	case "int":
		return `    printf("%d\n", __result);`
	case "float":
		return `    printf("%g\n", __result);`
	case "bool":
		return `    printf(__result ? "true\n" : "false\n");`
	case "string":
		return "    __printJSONString(__result);\n    putchar('\\n');"
	case "int[]":
		return "    putchar('[');\n    for (int i = 0; i < __returnSize; i++) { if (i) putchar(','); printf(\"%d\", __result[i]); }\n    putchar(']'); putchar('\\n');"
	case "float[]":
		return "    putchar('[');\n    for (int i = 0; i < __returnSize; i++) { if (i) putchar(','); printf(\"%g\", __result[i]); }\n    putchar(']'); putchar('\\n');"
	case "bool[]":
		return "    putchar('[');\n    for (int i = 0; i < __returnSize; i++) { if (i) putchar(','); printf(__result[i] ? \"true\" : \"false\"); }\n    putchar(']'); putchar('\\n');"
	case "string[]":
		return "    putchar('[');\n    for (int i = 0; i < __returnSize; i++) { if (i) putchar(','); __printJSONString(__result[i]); }\n    putchar(']'); putchar('\\n');"
	default:
		return ""
	}
}

func wrapC(sig Signature, userSource string) string {
	var declLines, callArgs []string
	for i, p := range sig.Params {
		decl, callExpr := cParseParam(i, p)
		declLines = append(declLines, decl)
		callArgs = append(callArgs, callExpr)
	}

	retDecl, isArrReturn := cReturnType(sig.ReturnType)
	callArgsStr := strings.Join(callArgs, ", ")
	var resultDecl string
	if isArrReturn {
		argsWithOut := callArgsStr
		if argsWithOut != "" {
			argsWithOut += ", "
		}
		argsWithOut += "&__returnSize"
		resultDecl = fmt.Sprintf("    int __returnSize;\n    %s __result = %s(%s);", retDecl, sig.FunctionName, argsWithOut)
	} else {
		resultDecl = fmt.Sprintf("    %s __result = %s(%s);", retDecl, sig.FunctionName, callArgsStr)
	}

	var b strings.Builder
	b.WriteString(userSource)
	b.WriteString("\n\n")
	b.WriteString(cHelpers)
	b.WriteString("\nint main() {\n")
	b.WriteString("    char __buf[65536];\n")
	b.WriteString("    int __len = fread(__buf, 1, sizeof(__buf) - 1, stdin);\n")
	b.WriteString("    __buf[__len] = '\\0';\n")
	b.WriteString("    char* __input = __trim(__buf);\n")
	b.WriteString("    int __ilen = strlen(__input);\n")
	b.WriteString("    char __inner[65536];\n")
	b.WriteString("    if (__ilen > 2) { strncpy(__inner, __input + 1, __ilen - 2); __inner[__ilen - 2] = '\\0'; } else { __inner[0] = '\\0'; }\n")
	b.WriteString("    __splitTop(__inner);\n")
	// __toks/__numToks get reused (clobbered) by any array-typed param's
	// own __parseXArray call below, so the top-level arg tokens have to be
	// copied out to a stable buffer before parsing any of them.
	b.WriteString("    char __args[__MAX_TOKENS][__MAX_TOKLEN];\n")
	b.WriteString("    for (int __i = 0; __i < __numToks; __i++) strcpy(__args[__i], __toks[__i]);\n\n")
	if len(declLines) > 0 {
		b.WriteString(strings.Join(declLines, "\n"))
		b.WriteString("\n\n")
	}
	b.WriteString(resultDecl)
	b.WriteString("\n\n")
	b.WriteString(cSerializeReturn(sig.ReturnType))
	b.WriteString("\n    return 0;\n}\n")
	return b.String()
}
