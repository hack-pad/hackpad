//go:build js
// +build js

package interop

import (
	"syscall/js"
)

var (
	jsReflectGet = js.Global().Get("Reflect").Get("get")
)

type cacher struct {
	cache map[string]js.Value
}

func (c *cacher) value(key string, valueFn func() interface{}) js.Value {
	if val, ok := c.cache[key]; ok {
		return val
	}
	if c.cache == nil {
		c.cache = make(map[string]js.Value)
	}
	val := js.ValueOf(valueFn())
	c.cache[key] = val
	return val
}

type StringCache struct {
	cacher
}

func (c *StringCache) Value(s string) js.Value {
	return c.value(s, identityStringGetter{s}.value)
}

func (c *StringCache) GetProperty(obj js.Value, key string) js.Value {
	jsKey := c.Value(key)
	return jsReflectGet.Invoke(obj, jsKey)
}

// CallCache caches a member function by name, then runs Invoke instead of Call.
// Has a slight performance boost, since it amortizes Reflect.get.
type CallCache struct {
	cacher
}

func (c *CallCache) Call(jsObj js.Value, s string, args ...interface{}) js.Value {
	valueFn := objGetter{key: s, obj: jsObj}.value
	return c.value(s, valueFn).Invoke(args...)
}

type identityStringGetter struct {
	s string
}

func (i identityStringGetter) value() interface{} {
	return i.s
}

type objGetter struct {
	key string
	obj js.Value
}

func (o objGetter) value() interface{} {
	return o.obj.Get(o.key).Call("bind", o.obj)
}
