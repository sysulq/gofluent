package task

import (
	"fmt"
	"reflect"
)

type NotCompletedStatus struct {}

var NotCompleted = &NotCompletedStatus {}

func (_ *NotCompletedStatus) Error() string { return "" }

type PanickedStatus struct {
	panic interface {}
}

func typeName(type_ reflect.Type) string {
	if type_.Kind() == reflect.Ptr {
		return "*" + typeName(type_.Elem())
	} else {
		return type_.Name()
	}
}

func (panicked *PanickedStatus) Error() string {
	switch panic_ := panicked.panic.(type) {
	case string:
		return panic_
	case error:
		return fmt.Sprintf("(%s) %s", typeName(reflect.TypeOf(panic_)), panic_.Error())
	default:
		type_ := reflect.TypeOf(panic_)
		method, ok := type_.MethodByName("String")
		if ok && method.Type.NumIn() == 1 {
			result := method.Func.Call([]reflect.Value { reflect.ValueOf(panic_) })
			if len(result) == 1 && result[0].Type().Kind() == reflect.String {
				return fmt.Sprintf("(%s) %s", typeName(type_), result[0].String())
			}
		}
		return fmt.Sprintf("(%s)", typeName(type_))
	}
}

type TaskStatus interface {
	Status() error
	Result() interface {}
	Poll()
}

type TaskRunner interface {
	Run(func () (interface {}, error)) (TaskStatus, error)
}

