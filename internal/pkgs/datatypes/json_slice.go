package datatypes

import (
	"gorm.io/datatypes"
)

// JSONSlice give a generic data type for json encoded slice data.
type JSONSlice[T any] = datatypes.JSONSlice[T]

func NewJSONSlice[T any](v []T) JSONSlice[T] {
	return datatypes.NewJSONSlice[T](v)
}
