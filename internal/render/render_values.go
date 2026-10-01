package render

import (
	"fmt"
	"strconv"
	"strings"
)

func yamlQuote(s string) string { return strconv.Quote(s) }
func yamlArray(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = yamlQuote(v)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
func writeTOMLString(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s = %s\n", key, strconv.Quote(value))
}
func writeTOMLArray(b *strings.Builder, key string, values []string) {
	fmt.Fprintf(b, "%s = %s\n", key, yamlArray(values))
}
func title(s string) string { return strings.ReplaceAll(s, "-", " ") }
