package harness

import (
	"fmt"
	"strings"
)

// javaHelpers is a fixed library of hand-rolled JSON parse/serialize
// helpers - Java has no JSON in its standard library. __serialize takes an
// Object (primitive return values auto-box, arrays are already objects) so
// only one serialization method is needed regardless of the concrete
// return type. Parsing still needs one function per concrete type since
// Java requires statically-typed local variables at the call site.
const javaHelpers = `    private static java.util.List<String> __splitTop(String s) {
        java.util.List<String> parts = new java.util.ArrayList<>();
        int depth = 0;
        boolean inStr = false;
        StringBuilder cur = new StringBuilder();
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            if (inStr) {
                cur.append(c);
                if (c == '\\' && i + 1 < s.length()) { cur.append(s.charAt(++i)); continue; }
                if (c == '"') inStr = false;
                continue;
            }
            if (c == '"') { inStr = true; cur.append(c); continue; }
            if (c == '[') depth++;
            if (c == ']') depth--;
            if (c == ',' && depth == 0) { parts.add(cur.toString()); cur.setLength(0); continue; }
            cur.append(c);
        }
        if (cur.length() > 0) parts.add(cur.toString());
        return parts;
    }
    private static String __parseString(String tok) {
        String t = tok.trim();
        t = t.substring(1, t.length() - 1);
        return t.replace("\\\"", "\"").replace("\\\\", "\\");
    }
    private static int[] __parseIntArray(String tok) {
        String inner = tok.trim();
        inner = inner.substring(1, inner.length() - 1);
        java.util.List<String> toks = __splitTop(inner);
        int[] out = new int[toks.size()];
        for (int i = 0; i < toks.size(); i++) out[i] = Integer.parseInt(toks.get(i).trim());
        return out;
    }
    private static double[] __parseDoubleArray(String tok) {
        String inner = tok.trim();
        inner = inner.substring(1, inner.length() - 1);
        java.util.List<String> toks = __splitTop(inner);
        double[] out = new double[toks.size()];
        for (int i = 0; i < toks.size(); i++) out[i] = Double.parseDouble(toks.get(i).trim());
        return out;
    }
    private static boolean[] __parseBoolArray(String tok) {
        String inner = tok.trim();
        inner = inner.substring(1, inner.length() - 1);
        java.util.List<String> toks = __splitTop(inner);
        boolean[] out = new boolean[toks.size()];
        for (int i = 0; i < toks.size(); i++) out[i] = toks.get(i).trim().equals("true");
        return out;
    }
    private static String[] __parseStringArray(String tok) {
        String inner = tok.trim();
        inner = inner.substring(1, inner.length() - 1);
        java.util.List<String> toks = __splitTop(inner);
        String[] out = new String[toks.size()];
        for (int i = 0; i < toks.size(); i++) out[i] = __parseString(toks.get(i));
        return out;
    }
    private static String __jsonString(String s) {
        return "\"" + s.replace("\\", "\\\\").replace("\"", "\\\"") + "\"";
    }
    private static String __serialize(Object o) {
        if (o instanceof int[]) {
            int[] a = (int[]) o;
            StringBuilder sb = new StringBuilder("[");
            for (int i = 0; i < a.length; i++) { if (i > 0) sb.append(","); sb.append(a[i]); }
            return sb.append("]").toString();
        }
        if (o instanceof double[]) {
            double[] a = (double[]) o;
            StringBuilder sb = new StringBuilder("[");
            for (int i = 0; i < a.length; i++) { if (i > 0) sb.append(","); sb.append(a[i]); }
            return sb.append("]").toString();
        }
        if (o instanceof boolean[]) {
            boolean[] a = (boolean[]) o;
            StringBuilder sb = new StringBuilder("[");
            for (int i = 0; i < a.length; i++) { if (i > 0) sb.append(","); sb.append(a[i]); }
            return sb.append("]").toString();
        }
        if (o instanceof String[]) {
            String[] a = (String[]) o;
            StringBuilder sb = new StringBuilder("[");
            for (int i = 0; i < a.length; i++) { if (i > 0) sb.append(","); sb.append(__jsonString(a[i])); }
            return sb.append("]").toString();
        }
        if (o instanceof String) return __jsonString((String) o);
        if (o instanceof Boolean) return ((Boolean) o) ? "true" : "false";
        return String.valueOf(o);
    }
`

func javaType(t string) string {
	if isArray(t) {
		return javaType(elemType(t)) + "[]"
	}
	switch t {
	case "int":
		return "int"
	case "float":
		return "double"
	case "bool":
		return "boolean"
	default:
		return "String"
	}
}

func javaParseExpr(t, tokExpr string) string {
	switch t {
	case "int":
		return fmt.Sprintf("Integer.parseInt(%s.trim())", tokExpr)
	case "float":
		return fmt.Sprintf("Double.parseDouble(%s.trim())", tokExpr)
	case "bool":
		return fmt.Sprintf("%s.trim().equals(\"true\")", tokExpr)
	case "string":
		return fmt.Sprintf("__parseString(%s)", tokExpr)
	case "int[]":
		return fmt.Sprintf("__parseIntArray(%s)", tokExpr)
	case "float[]":
		return fmt.Sprintf("__parseDoubleArray(%s)", tokExpr)
	case "bool[]":
		return fmt.Sprintf("__parseBoolArray(%s)", tokExpr)
	case "string[]":
		return fmt.Sprintf("__parseStringArray(%s)", tokExpr)
	default:
		return tokExpr
	}
}

func stubJava(sig Signature) string {
	var params []string
	for _, p := range sig.Params {
		params = append(params, fmt.Sprintf("%s %s", javaType(p.Type), p.Name))
	}
	return fmt.Sprintf("class Solution {\n    public %s %s(%s) {\n        \n    }\n}\n",
		javaType(sig.ReturnType), sig.FunctionName, strings.Join(params, ", "))
}

// extractLeadingImports pulls any `import ...;` lines off the front of the
// user's source - Java requires all imports to precede all type
// declarations in the file (same rule as Go), and wrapJava puts the
// generated Main class before the user's Solution class, so any imports
// the user wrote have to be hoisted above Main too.
func extractLeadingImports(source string) (imports []string, rest string) {
	lines := strings.Split(source, "\n")
	i := 0
	for ; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "import ") {
			imports = append(imports, trimmed)
			continue
		}
		break
	}
	return imports, strings.Join(lines[i:], "\n")
}

// wrapJava puts the generated `public class Main` FIRST in the file -
// Piston's Java runner picks whichever class is declared first as the
// entry point regardless of which one is public, so the user's
// `class Solution` must come after it, not before.
func wrapJava(sig Signature, userSource string) string {
	imports, rest := extractLeadingImports(userSource)

	var decls, callArgs []string
	for i, p := range sig.Params {
		varName := fmt.Sprintf("__arg%d", i)
		tokExpr := fmt.Sprintf("__toks.get(%d)", i)
		decls = append(decls, fmt.Sprintf("        %s %s = %s;", javaType(p.Type), varName, javaParseExpr(p.Type, tokExpr)))
		callArgs = append(callArgs, varName)
	}

	var b strings.Builder
	if len(imports) > 0 {
		b.WriteString(strings.Join(imports, "\n"))
		b.WriteString("\n\n")
	}
	b.WriteString("public class Main {\n")
	b.WriteString("    public static void main(String[] args) throws Exception {\n")
	b.WriteString("        StringBuilder __sb = new StringBuilder();\n")
	b.WriteString("        java.util.Scanner __sc = new java.util.Scanner(System.in);\n")
	b.WriteString("        while (__sc.hasNextLine()) { __sb.append(__sc.nextLine()); }\n")
	b.WriteString("        String __input = __sb.toString().trim();\n")
	b.WriteString("        String __inner = __input.length() > 2 ? __input.substring(1, __input.length() - 1) : \"\";\n")
	b.WriteString("        java.util.List<String> __toks = __splitTop(__inner);\n\n")
	if len(decls) > 0 {
		b.WriteString(strings.Join(decls, "\n"))
		b.WriteString("\n\n")
	}
	b.WriteString("        Solution __sol = new Solution();\n")
	fmt.Fprintf(&b, "        Object __result = __sol.%s(%s);\n", sig.FunctionName, strings.Join(callArgs, ", "))
	b.WriteString("        System.out.println(__serialize(__result));\n")
	b.WriteString("    }\n\n")
	b.WriteString(javaHelpers)
	b.WriteString("}\n\n")
	b.WriteString(rest)
	return b.String()
}
