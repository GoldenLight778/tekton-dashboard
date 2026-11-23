package repository

import (
	"strconv"
	"strings"
)

// Rebind converts '?' placeholders to '$1, $2, ...' when the driver is postgres/postgresql.
// For sqlite and mysql, it returns the query unchanged.
func Rebind(driver, query string) string {
	if driver != "postgres" && driver != "postgresql" && driver != "pgx" {
		return query
	}

	var b strings.Builder
	b.Grow(len(query) + 16)

	paramIdx := 1
	inSingleQuote := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			inSingleQuote = !inSingleQuote
			b.WriteByte(ch)
		} else if ch == '?' && !inSingleQuote {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(paramIdx))
			paramIdx++
		} else {
			b.WriteByte(ch)
		}
	}
	return b.String()
}
