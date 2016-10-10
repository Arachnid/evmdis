package evmdis

import (
	"reflect"
)

type TypeMap struct {
	data map[reflect.Type]interface{}
}

func (self *TypeMap) Get(obj interface{}) {
	element := reflect.ValueOf(obj).Elem()
	if value, ok := self.data[element.Type()]; ok {
		element.Set(reflect.ValueOf(value))
	} else {
		element.Set(reflect.Zero(element.Type()))
	}
}

func (self *TypeMap) Pop(obj interface{}) {
	element := reflect.ValueOf(obj).Elem()
	if value, ok := self.data[element.Type()]; ok {
		element.Set(reflect.ValueOf(value))
		delete(self.data, element.Type())
	} else {
		element.Set(reflect.Zero(element.Type()))
	}
}

func (self *TypeMap) Set(obj interface{}) {
	element := reflect.ValueOf(obj).Elem()
	self.data[element.Type()] = element.Interface()
}

func NewTypeMap() *TypeMap {
	return &TypeMap{make(map[reflect.Type]interface{})}
}
