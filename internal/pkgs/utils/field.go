package utils

type Field struct {
	Key   string
	Value interface{}
}

var WithField = func(k string, v interface{}) Field {
	return Field{Key: k, Value: v}
}
