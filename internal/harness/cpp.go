package harness

import (
	"fmt"
	"strings"
)

// cppHelpers is a fixed hand-rolled JSON parse/serialize library - C++ has
// no JSON in its standard library. Placed after the user's function and
// before main(), so no forward declarations are needed. #include ordering
// has no constraint in C/C++ (unlike Go/Java's import rules), so unlike
// wrapGo/wrapJava this can simply be appended after the user's source.
const cppHelpers = `
vector<string> __splitTop(const string& s) {
    vector<string> parts;
    int depth = 0;
    bool inStr = false;
    string cur;
    for (size_t i = 0; i < s.size(); i++) {
        char c = s[i];
        if (inStr) {
            cur += c;
            if (c == '\\' && i + 1 < s.size()) { cur += s[++i]; continue; }
            if (c == '"') inStr = false;
            continue;
        }
        if (c == '"') { inStr = true; cur += c; continue; }
        if (c == '[') depth++;
        if (c == ']') depth--;
        if (c == ',' && depth == 0) { parts.push_back(cur); cur.clear(); continue; }
        cur += c;
    }
    if (!cur.empty()) parts.push_back(cur);
    return parts;
}
string __trim(const string& s) {
    size_t a = s.find_first_not_of(" \t\n\r");
    if (a == string::npos) return "";
    size_t b = s.find_last_not_of(" \t\n\r");
    return s.substr(a, b - a + 1);
}
string __innerOf(const string& tok) {
    string t = __trim(tok);
    if (t.size() < 2) return "";
    return t.substr(1, t.size() - 2);
}
string __parseStr(const string& tok) {
    string t = __innerOf(tok);
    string out;
    for (size_t i = 0; i < t.size(); i++) {
        if (t[i] == '\\' && i + 1 < t.size()) { out += t[++i]; }
        else out += t[i];
    }
    return out;
}
vector<int> __parseIntArr(const string& tok) {
    vector<string> toks = __splitTop(__innerOf(tok));
    vector<int> out;
    for (auto& x : toks) out.push_back(stoi(__trim(x)));
    return out;
}
vector<double> __parseDoubleArr(const string& tok) {
    vector<string> toks = __splitTop(__innerOf(tok));
    vector<double> out;
    for (auto& x : toks) out.push_back(stod(__trim(x)));
    return out;
}
vector<bool> __parseBoolArr(const string& tok) {
    vector<string> toks = __splitTop(__innerOf(tok));
    vector<bool> out;
    for (auto& x : toks) out.push_back(__trim(x) == "true");
    return out;
}
vector<string> __parseStringArr(const string& tok) {
    vector<string> toks = __splitTop(__innerOf(tok));
    vector<string> out;
    for (auto& x : toks) out.push_back(__parseStr(x));
    return out;
}
string __jsonStr(const string& s) {
    string out = "\"";
    for (char c : s) { if (c == '"' || c == '\\') out += '\\'; out += c; }
    out += "\"";
    return out;
}
string __joinNum(const vector<int>& a) {
    string out = "[";
    for (size_t i = 0; i < a.size(); i++) { if (i) out += ","; out += to_string(a[i]); }
    return out + "]";
}
string __joinNum(const vector<double>& a) {
    string out = "[";
    for (size_t i = 0; i < a.size(); i++) { if (i) out += ","; out += to_string(a[i]); }
    return out + "]";
}
string __joinBool(const vector<bool>& a) {
    string out = "[";
    for (size_t i = 0; i < a.size(); i++) { if (i) out += ","; out += (a[i] ? "true" : "false"); }
    return out + "]";
}
string __joinStr(const vector<string>& a) {
    string out = "[";
    for (size_t i = 0; i < a.size(); i++) { if (i) out += ","; out += __jsonStr(a[i]); }
    return out + "]";
}
`

func cppType(t string) string {
	if isArray(t) {
		return "vector<" + cppType(elemType(t)) + ">"
	}
	switch t {
	case "int":
		return "int"
	case "float":
		return "double"
	case "bool":
		return "bool"
	default:
		return "string"
	}
}

func cppParseExpr(t, tokExpr string) string {
	switch t {
	case "int":
		return fmt.Sprintf("stoi(__trim(%s))", tokExpr)
	case "float":
		return fmt.Sprintf("stod(__trim(%s))", tokExpr)
	case "bool":
		return fmt.Sprintf("__trim(%s) == \"true\"", tokExpr)
	case "string":
		return fmt.Sprintf("__parseStr(%s)", tokExpr)
	case "int[]":
		return fmt.Sprintf("__parseIntArr(%s)", tokExpr)
	case "float[]":
		return fmt.Sprintf("__parseDoubleArr(%s)", tokExpr)
	case "bool[]":
		return fmt.Sprintf("__parseBoolArr(%s)", tokExpr)
	case "string[]":
		return fmt.Sprintf("__parseStringArr(%s)", tokExpr)
	default:
		return tokExpr
	}
}

func cppSerializeExpr(t, resultExpr string) string {
	switch t {
	case "bool":
		return fmt.Sprintf("(%s ? \"true\" : \"false\")", resultExpr)
	case "string":
		return fmt.Sprintf("__jsonStr(%s)", resultExpr)
	case "int[]", "float[]":
		return fmt.Sprintf("__joinNum(%s)", resultExpr)
	case "bool[]":
		return fmt.Sprintf("__joinBool(%s)", resultExpr)
	case "string[]":
		return fmt.Sprintf("__joinStr(%s)", resultExpr)
	default: // int, float
		return resultExpr
	}
}

// stubCpp pulls in <bits/stdc++.h> (a GCC convenience header covering the
// whole standard library) so the user never has to think about includes
// for typical LeetCode-style solutions - this also sidesteps the
// import-ordering problems wrapGo/wrapJava have to work around, since
// #include has no such ordering constraint.
func stubCpp(sig Signature) string {
	var params []string
	for _, p := range sig.Params {
		params = append(params, fmt.Sprintf("%s %s", cppType(p.Type), p.Name))
	}
	return fmt.Sprintf("#include <bits/stdc++.h>\nusing namespace std;\n\n%s %s(%s) {\n    \n}\n",
		cppType(sig.ReturnType), sig.FunctionName, strings.Join(params, ", "))
}

func wrapCpp(sig Signature, userSource string) string {
	var decls, callArgs []string
	for i, p := range sig.Params {
		varName := fmt.Sprintf("__arg%d", i)
		tokExpr := fmt.Sprintf("__toks[%d]", i)
		decls = append(decls, fmt.Sprintf("    auto %s = %s;", varName, cppParseExpr(p.Type, tokExpr)))
		callArgs = append(callArgs, varName)
	}

	var b strings.Builder
	b.WriteString(userSource)
	b.WriteString("\n\n")
	b.WriteString(cppHelpers)
	b.WriteString("\nint main() {\n")
	b.WriteString("    string __input((istreambuf_iterator<char>(cin)), istreambuf_iterator<char>());\n")
	b.WriteString("    __input = __trim(__input);\n")
	b.WriteString("    string __inner = __input.size() > 2 ? __input.substr(1, __input.size() - 2) : \"\";\n")
	b.WriteString("    vector<string> __toks = __splitTop(__inner);\n\n")
	if len(decls) > 0 {
		b.WriteString(strings.Join(decls, "\n"))
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "    auto __result = %s(%s);\n", sig.FunctionName, strings.Join(callArgs, ", "))
	fmt.Fprintf(&b, "    cout << %s << endl;\n", cppSerializeExpr(sig.ReturnType, "__result"))
	b.WriteString("    return 0;\n}\n")
	return b.String()
}
