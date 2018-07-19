package filter

import (
	"strconv"
)

type Numeric struct {
	*Common
}

func (filter Numeric) ParseValue(value interface{}) string {
	uint64Value, canCast := value.(uint64)
	if canCast {
		return strconv.FormatUint(uint64Value, 10)
	}

	int64Value, canCast := value.(int64)
	if canCast {
		return strconv.FormatInt(int64Value, 10)
	}

	stringValue, canCast := value.(string)
	if canCast {
		intValue, _ := strconv.ParseInt(stringValue, 10, 64)
		return strconv.FormatInt(intValue, 10)
	}

	panic("todo - cannot parse value!")
}
