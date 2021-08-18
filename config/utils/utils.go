package utils

import "reflect"

func IsBlank(str string) bool {
	stringRune := []rune(str)
	if len(stringRune) == 0 {
		return true
	}
	for _, r := range stringRune {
		if r != ' ' {
			return false
		}
	}
	return true
}

func NotBlank(str string) bool {
	return !IsBlank(str)
}

func GetDefaultValidators() map[reflect.Kind]func(data interface{}) bool {
	validators := make(map[reflect.Kind]func(data interface{}) bool)
	validators[reflect.String] = func(data interface{}) bool {
		str := data.(string)
		return !IsBlank(str)
	}
	validators[reflect.Uint] = func(data interface{}) bool {
		return true
	}
	validators[reflect.Int] = func(data interface{}) bool {
		return true
	}
	validators[reflect.Float64] = func(data interface{}) bool {
		return true
	}
	validators[reflect.Bool] = func(data interface{}) bool {
		return true
	}
	return validators
}
