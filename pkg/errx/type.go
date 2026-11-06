package errx

import (
	"strconv"
	"sync"
)

var (
	// registryMu 串行化对关联包级状态的访问。
	registryMu sync.Mutex

	// registeredIDs 保存当前实现使用的条目。
	registeredIDs = map[int]string{}

	// registered 保存当前实现使用的条目。
	registered = map[string]int{}
)

// Type 表示一类稳定可判别的共享业务错误。
type Type struct {
	// name 保存注册时指定的错误类型名称。
	name string

	// code 保存与名称一一对应的正整数错误码。
	code int
}

// NewType 注册一个新的错误类型，并校验 name/code 在进程内唯一。
//
// 要做什么：
// - 建立名称与正整数错误码一一对应的进程级错误类型。
//
// 具体怎么做：
// - 拒绝空名称和非正数编码，再持有注册锁检查两个索引是否已有对应项。
// - 检查通过后同时登记名称和编码；非法参数或重复注册均触发 panic。
//
// 输出什么：
// - 返回保存该名称与编码的错误类型对象。
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
//
// 要做什么：
// - 读取错误类型名称，并允许对空类型指针查询。
//
// 具体怎么做：
// - 接收者为空时返回空字符串，否则读取注册时保存的名称。
//
// 输出什么：
// - 返回稳定名称；空接收者返回空字符串。
func (t *Type) Name() string {
	if t == nil {
		return ""
	}
	return t.name
}

// Code 返回错误类型的稳定数值编码。
//
// 要做什么：
// - 读取错误类型编码，并允许对空类型指针查询。
//
// 具体怎么做：
// - 接收者为空时返回零，否则读取注册时保存的编码。
//
// 输出什么：
// - 返回注册的正整数错误码；空接收者返回零。
func (t *Type) Code() int {
	if t == nil {
		return 0
	}
	return t.code
}
