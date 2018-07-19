package filter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tableaux-project/tableaux"
)

type RegexString struct {
	*Common
}

func (filter RegexString) ParseValue(value interface{}) string {
	stringVal, canCast := value.(string)
	if canCast {
		return fmt.Sprintf(`'%s'`, strings.Replace(stringVal, ".*", "%", -1))
	}

	panic("todo - cannot parse value!")
}

func (filter RegexString) Operator(value interface{}, filterMode tableaux.FilterMode) (Operator, error) {
	stringVal, canCast := value.(string)
	if !canCast {
		return "", errors.New("cannot cast to string")
	}

	if strings.Contains(stringVal, ".*") {
		switch filterMode {
		case tableaux.FilterEquals:
			return OperatorLike, nil
		case tableaux.FilterNotEquals:
			return OperatorNotLike, nil
		default:
			return filter.Common.Operator(value, filterMode)
		}
	} else {
		return filter.Common.Operator(value, filterMode)
	}
}
