package filter

import (
	"fmt"
)

type PlainString struct {
	*Common
}

func (filter PlainString) ParseValue(value interface{}) string {
	stringVal, canCast := value.(string)
	if canCast {
		return fmt.Sprintf(`'%s'`, stringVal)
	}

	panic("todo - cannot parse value!")
}
