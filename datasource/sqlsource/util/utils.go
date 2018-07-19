package util

import (
	"database/sql"
	"regexp"
	"strings"

	"gopkg.in/birkirb/loggers.v1/log"
)

func DescriptorToIdentifier(descriptor string) string {
	re := regexp.MustCompile("([A-Z].+?)")
	rest := re.ReplaceAllString(descriptor, "_${1}")
	return strings.ToLower(rest)
}

func IdentifierToDescriptor(database string) string {
	re := regexp.MustCompile("(_[a-z])")
	return re.ReplaceAllStringFunc(database, func(s string) string {
		return strings.ToUpper(strings.TrimPrefix(s, "_"))
	})
}

// LoggingRowsCloser is a helper method which wraps row closing
// with a logging statement, if an error ocurs.
func LoggingRowsCloser(rows *sql.Rows, usage string) {
	if err := rows.Close(); err != nil {
		log.WithFields(
			"error", err,
			"usage", usage,
		).Error("Failed to explicitly close rows")
	}
}
