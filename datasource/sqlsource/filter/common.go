package filter

import (
	"fmt"

	"github.com/tableaux-project/tableaux"
)

type Common struct {
}

func (filter Common) Operator(_ interface{}, filterMode tableaux.FilterMode) (Operator, error) {
	switch filterMode {
	case tableaux.FilterEquals:
		return OperatorEqual, nil
	case tableaux.FilterGreater:
		return OperatorGreater, nil
	case tableaux.FilterGreaterEquals:
		return OperatorGreaterEquals, nil
	case tableaux.FilterLesser:
		return OperatorLesser, nil
	case tableaux.FilterLesserEquals:
		return OperatorLesserEquals, nil
	case tableaux.FilterNotEquals:
		return OperatorNotEqual, nil
	default:
		return "", fmt.Errorf("unknown filter mode %s", filterMode)
	}
}
