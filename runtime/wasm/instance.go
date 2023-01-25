package wasm

import (
	"C"

	"fmt"
	"reflect"
	"unsafe"
)

// ---
// Docs:
// Wasm Types: https://webassembly.github.io/spec/core/syntax/types.html#
// Go   Types: https://go.dev/ref/spec#Types
//
// WASM_ANYREF = 0x80
// WASM_FUNCREF = 0x81
// WASM_I32 = 0x0
// WASM_I64 = 0x1
// WASM_F32 = 0x2
// WASM_F64 = 0x3
//
// ---

// Call execute the wasm instance and return
func (instance *Instance) Call(method string, res interface{}, args ...interface{}) error {

	offsets, err := instance.wArgs(args)
	defer instance.free(offsets)

	if err != nil {
		return err
	}

	returns := []interface{}{nil}
	err = instance.wamrInstance.CallFuncV(method, 1, returns, args...)
	if err != nil {
		return err
	}

	err = instance.wRes(returns[0], res)
	if err != nil {
		return err
	}

	return nil
}

func (instance *Instance) wRes(value interface{}, res interface{}) error {
	if res == nil {
		return nil
	}
	ptr := reflect.ValueOf(res)
	typ := ptr.Type()
	if typ.Kind() != reflect.Pointer {
		return fmt.Errorf("res must be a pointer")
	}

	var err error
	var rval reflect.Value
	switch ptr.Elem().Kind() {
	case reflect.Int:
		rval, err = instance.wResInt(value)
		if err != nil {
			return err
		}
		break

	case reflect.String:
		rval, err = instance.wResString(value)
		if err != nil {
			return err
		}
		break

	case reflect.Slice:

		if _, ok := ptr.Elem().Interface().([]byte); !ok {
			return fmt.Errorf("%s type does not support", ptr.Elem().Kind().String())
		}

		rval, err = instance.wResBytes(value)
		if err != nil {
			return err
		}
		break

	default:
		return fmt.Errorf("%s type does not support", ptr.Elem().Kind().String())
	}

	ptr.Elem().Set(rval)
	return nil
}

func (instance *Instance) wResInt(value interface{}) (reflect.Value, error) {

	switch v := value.(type) {
	case int32:
		return reflect.ValueOf(int(v)), nil
	}

	return reflect.Value{}, fmt.Errorf("Return type does not support")
}

func (instance *Instance) wResString(value interface{}) (reflect.Value, error) {

	switch v := value.(type) {
	case int32:
		addr := instance.wamrInstance.AddrAppToNative(uint32(v))
		ptr := (*C.char)(unsafe.Pointer(addr))
		return reflect.ValueOf(C.GoString(ptr)), nil
	}

	return reflect.Value{}, fmt.Errorf("Return type does not support")
}

func (instance *Instance) wResBytes(value interface{}) (reflect.Value, error) {

	switch v := value.(type) {
	case int32:
		addr := instance.wamrInstance.AddrAppToNative(uint32(v))
		val := C.GoString((*C.char)(unsafe.Pointer(addr)))
		return reflect.ValueOf([]byte(val)), nil
	}
	return reflect.Value{}, fmt.Errorf("Return type does not support")
}

func (instance *Instance) wArgs(args []interface{}) ([]uint32, error) {

	offsets := []uint32{}
	if args == nil {
		args = []interface{}{}
		return offsets, nil
	}

	for i := range args {
		v, offset, err := instance.wArg(args[i])
		if err != nil {
			return offsets, err
		}

		if offset != 0 {
			offsets = append(offsets, offset)
		}
		args[i] = v
	}

	return offsets, nil
}

func (instance *Instance) wArg(arg interface{}) (interface{}, uint32, error) {

	if instance.isNumber(arg) {
		v, err := instance.wArgNumber(arg)
		return v, 0, err
	}

	if instance.isBool(arg) {
		v, err := instance.wArgBool(arg)
		return v, 0, err
	}

	switch v := arg.(type) {
	case string:
		return instance.wArgString(v)

	case []byte:
		return instance.wArgBytes(v)
	}

	return nil, 0, nil
}

// Golang numbers
// uint        either 32 o 64 bits
// int         same size as uint
// uintptr     an unsigned integer large enough to store the uninterpreted bits of a pointer value

// uint8       the set of all unsigned  8-bit integers (0 to 255)
// uint16      the set of all unsigned 16-bit integers (0 to 65535)
// uint32      the set of all unsigned 32-bit integers (0 to 4294967295)
// uint64      the set of all unsigned 64-bit integers (0 to 18446744073709551615)

// int8        the set of all signed  8-bit integers (-128 to 127)
// int16       the set of all signed 16-bit integers (-32768 to 32767)
// int32       the set of all signed 32-bit integers (-2147483648 to 2147483647)
// int64       the set of all signed 64-bit integers (-9223372036854775808 to 9223372036854775807)

// float32     the set of all IEEE-754 32-bit floating-point numbers
// float64     the set of all IEEE-754 64-bit floating-point numbers
func (instance *Instance) wArgNumber(arg interface{}) (interface{}, error) {
	switch v := arg.(type) {
	case int32, int64, float32, float64:
		return v, nil
	case uint8:
		return int32(v), nil
	case uint16:
		return int32(v), nil
	case uint32:
		return int32(v), nil
	case uint64:
		return int64(v), nil
	case int8:
		return int32(v), nil
	case int16:
		return int32(v), nil
	case int:
		return int32(v), nil
	}
	return nil, fmt.Errorf("error: %#v is not a type of number", arg)
}

func (instance *Instance) wArgBool(arg interface{}) (interface{}, error) {
	v, ok := arg.(bool)
	if !ok {
		return nil, fmt.Errorf("error: %#v is not a type of boolean", arg)
	}
	if v {
		return int32(1), nil
	}
	return int32(1), nil
}

// byte        alias for uint8
// rune        alias for int32
func (instance *Instance) wArgString(arg string) (interface{}, uint32, error) {
	cstr := C.CBytes([]byte(arg))
	offset, addr := instance.wamrInstance.ModuleMalloc(uint32(len(arg) + int(unsafe.Sizeof(arg))))
	ptr := (*[]byte)(unsafe.Pointer(addr))
	cptr := (*[]byte)(unsafe.Pointer(cstr))
	*ptr = *cptr
	return int32(offset), offset, nil
}

// byte        alias for uint8
// rune        alias for int32
func (instance *Instance) wArgBytes(arg []byte) (interface{}, uint32, error) {
	offset, addr := instance.wamrInstance.ModuleMalloc(uint32(len(arg) + int(unsafe.Sizeof(arg))))
	bytes := C.CBytes(arg)
	ptr := (*[]byte)(unsafe.Pointer(addr))
	cptr := (*[]byte)(unsafe.Pointer(bytes))
	*ptr = *cptr
	return int32(offset), offset, nil
}

func (instance *Instance) wArgJSON(arg interface{}) (interface{}, error) {
	return nil, nil
}

func (instance *Instance) free(offsets []uint32) {
	for _, offset := range offsets {
		instance.wamrInstance.ModuleFree(offset)
	}
}

func (instance *Instance) isNumber(v interface{}) bool {
	switch v.(type) {
	case int, int32, int64, float32, float64, uint8, uint16, uint32, uint64, int8, int16:
		return true
	}
	return false
}

func (instance *Instance) isBool(v interface{}) bool {
	_, ok := v.(bool)
	return ok
}
