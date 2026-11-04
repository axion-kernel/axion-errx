package errx

import (
	"strconv"
	"sync"
)

// Type 表示一类稳定可判别的共享业务错误。
type Type struct {
	name string
	code int
}

var (
	registryMu    sync.Mutex
	registeredIDs = map[int]string{}
	registered    = map[string]int{}
)

// NewType 注册一个新的错误类型，并校验 name/code 在进程内唯一。
func NewType(name string, code int) *Type {
	if name == "" {
		panic("errx: type name is required")
	}
	if code <= 0 {
		panic("errx: type code must be > 0")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if existingName, ok := registeredIDs[code]; ok {
		panic("errx: duplicate type code " + name + " conflicts with " + existingName)
	}
	if existingCode, ok := registered[name]; ok {
		panic("errx: duplicate type name " + name + " conflicts with code " + strconv.Itoa(existingCode))
	}

	registeredIDs[code] = name
	registered[name] = code
	return &Type{name: name, code: code}
}

// Name 返回错误类型注册时声明的稳定名称。
func (t *Type) Name() string {
	if t == nil {
		return ""
	}
	return t.name
}

// Code 返回错误类型的稳定数值编码。
func (t *Type) Code() int {
	if t == nil {
		return 0
	}
	return t.code
}
