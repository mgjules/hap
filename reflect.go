package hap

import (
	"reflect"
	"runtime"
)

// GetFunctionName returns the function name.
func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
