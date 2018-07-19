package filter

import (
	"github.com/tableaux-project/tableaux"
)

type Operator string

const (
	OperatorLike          Operator = "LIKE"
	OperatorNotLike       Operator = "NOT LIKE"
	OperatorEqual         Operator = "="
	OperatorNotEqual      Operator = "!="
	OperatorGreater       Operator = "GREATER"
	OperatorGreaterEquals Operator = "GREATER_EQUALS"
	OperatorLesser        Operator = "LESSER"
	OperatorLesserEquals  Operator = "LESSER_EQUALS"
)

type Filter interface {
	ParseValue(value interface{}) string
	Operator(value interface{}, filterMode tableaux.FilterMode) (Operator, error)
}
