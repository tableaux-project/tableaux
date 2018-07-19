package filter

import (
	"strings"
)

type Boolean struct {
	*Common
}

func (filter Boolean) ParseValue(value interface{}) string {
	boolean, canCast := value.(bool)
	if canCast {
		if boolean {
			return "true"
		}

		return "false"
	}

	booleanString, canCast := value.(string)
	if canCast {
		boolean := booleanString == "1" || strings.ToLower(booleanString) == "true"

		if boolean {
			return "true"
		}

		return "false"
	}

	panic("todo - cannot parse value!")
}
