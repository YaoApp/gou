/*
 * Copyright (C) 2019 Intel Corporation.  All rights reserved.
 * SPDX-License-Identifier: Apache-2.0 WITH LLVM-exception
 */

package wamr

/*
#include <stdlib.h>
#include <wasm_export.h>

static inline void
PUT_I64_TO_ADDR(uint32_t *addr, int64_t value)
{
    union {
        int64_t val;
        uint32_t parts[2];
    } u;
    u.val = value;
    addr[0] = u.parts[0];
    addr[1] = u.parts[1];
}

static inline void
PUT_F64_TO_ADDR(uint32_t *addr, double value)
{
    union {
        double val;
        uint32_t parts[2];
    } u;
    u.val = value;
    addr[0] = u.parts[0];
    addr[1] = u.parts[1];
}

static inline int64_t
GET_I64_FROM_ADDR(uint32_t *addr)
{
    union {
        int64_t val;
        uint32_t parts[2];
    } u;
    u.parts[0] = addr[0];
    u.parts[1] = addr[1];
    return u.val;
}

static inline double
GET_F64_FROM_ADDR(uint32_t *addr)
{
    union {
        double val;
        uint32_t parts[2];
    } u;
    u.parts[0] = addr[0];
    u.parts[1] = addr[1];
    return u.val;
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// Instance the instance struct
type Instance struct {
	_instance     C.wasm_module_inst_t
	_execEnv      C.wasm_exec_env_t
	_module       *Module
	_exportsCache map[string]C.wasm_function_inst_t
}

// NewInstance Create instance from the module
func NewInstance(module *Module,
	stackSize uint, heapSize uint) (*Instance, error) {
	if module == nil {
		return nil, fmt.Errorf("NewInstance error: invalid input")
	}

	errorBytes := make([]byte, 128)
	errorPtr := (*C.char)(unsafe.Pointer(&errorBytes[0]))
	errorLen := C.uint(len(errorBytes))

	instance := C.wasm_runtime_instantiate(module.module, C.uint(stackSize),
		C.uint(heapSize), errorPtr, errorLen)
	if instance == nil {
		return nil, fmt.Errorf("NewInstance Error: %s", string(errorBytes))
	}

	_execEnv := C.wasm_runtime_create_exec_env(instance, C.uint(stackSize))
	if _execEnv == nil {
		C.wasm_runtime_deinstantiate(instance)
		return nil, fmt.Errorf("NewInstance Error: create _execEnv failed")
	}

	inst := &Instance{
		_instance:     instance,
		_execEnv:      _execEnv,
		_module:       module,
		_exportsCache: make(map[string]C.wasm_function_inst_t),
	}

	runtime.SetFinalizer(inst, func(inst *Instance) {
		inst.Destroy()
	})

	return inst, nil
}

// Destroy the instance
func (inst *Instance) Destroy() {
	runtime.SetFinalizer(inst, nil)
	if inst._instance != nil {
		C.wasm_runtime_deinstantiate(inst._instance)
	}
	if inst._execEnv != nil {
		C.wasm_runtime_destroy_exec_env(inst._execEnv)
	}
}

// CallFunc Call the wasm function with argument in the uint32 array, and store
// the return values back into the array
func (inst *Instance) CallFunc(funcName string,
	argc uint32, args []uint32) error {
	_func := inst._exportsCache[funcName]
	if _func == nil {
		cName := C.CString(funcName)
		defer C.free(unsafe.Pointer(cName))

		_func = C.wasm_runtime_lookup_function(inst._instance,
			cName, (*C.char)(C.NULL))
		if _func == nil {
			return fmt.Errorf("CallFunc error: lookup function failed")
		}
		inst._exportsCache[funcName] = _func
	}

	threadEnvInited := Runtime().ThreadEnvInited()
	if !threadEnvInited {
		Runtime().InitThreadEnv()
	}

	var argsC *C.uint32_t
	if argc > 0 {
		argsC = (*C.uint32_t)(unsafe.Pointer(&args[0]))
	}

	if !C.wasm_runtime_call_wasm(inst._execEnv, _func,
		C.uint(argc), argsC) {
		if !threadEnvInited {
			Runtime().DestroyThreadEnv()
		}
		return fmt.Errorf("CallFunc error: %s", string(inst.GetException()))
	}

	if !threadEnvInited {
		Runtime().DestroyThreadEnv()
	}
	return nil
}

// CallFuncV Call the wasm function with variant arguments, and store the return
// values back into the results array
func (inst *Instance) CallFuncV(funcName string,
	numResults uint32, results []interface{},
	args ...interface{}) error {
	_func := inst._exportsCache[funcName]
	if _func == nil {
		cName := C.CString(funcName)
		defer C.free(unsafe.Pointer(cName))

		_func = C.wasm_runtime_lookup_function(inst._instance,
			cName, (*C.char)(C.NULL))
		if _func == nil {
			return fmt.Errorf("CallFunc error: lookup function failed")
		}
		inst._exportsCache[funcName] = _func
	}

	paramCount := uint32(C.wasm_func_get_param_count(_func, inst._instance))
	resultCount := uint32(C.wasm_func_get_result_count(_func, inst._instance))

	if numResults < resultCount {
		str := "CallFunc error: invalid result count %d, " +
			"must be no smaller than %d"
		return fmt.Errorf(str, numResults, resultCount)
	}

	paramTypes := make([]C.uchar, paramCount, paramCount)
	resultTypes := make([]C.uchar, resultCount, resultCount)
	if paramCount > 0 {
		C.wasm_func_get_param_types(_func, inst._instance,
			(*C.uchar)(unsafe.Pointer(&paramTypes[0])))
	}
	if resultCount > 0 {
		C.wasm_func_get_result_types(_func, inst._instance,
			(*C.uchar)(unsafe.Pointer(&resultTypes[0])))
	}

	argvSize := paramCount * 2
	if resultCount > paramCount {
		argvSize = resultCount * 2
	}
	argv := make([]uint32, argvSize, argvSize)

	var i, argc uint32
	for _, arg := range args {
		if i >= paramCount {
			break
		}
		switch arg.(type) {
		case int32:
			if paramTypes[i] != C.WASM_I32 &&
				paramTypes[i] != C.WASM_FUNCREF &&
				paramTypes[i] != C.WASM_ANYREF {
				str := "CallFunc error: invalid param type %d, " +
					"expect i32 but got other"
				return fmt.Errorf(str, paramTypes[i])
			}
			argv[argc] = (uint32)(arg.(int32))
			argc++
			break
		case int64:
			if paramTypes[i] != C.WASM_I64 {
				str := "CallFunc error: invalid param type %d, " +
					"expect i64 but got other"
				return fmt.Errorf(str, paramTypes[i])
			}
			addr := (*C.uint32_t)(unsafe.Pointer(&argv[argc]))
			C.PUT_I64_TO_ADDR(addr, (C.int64_t)(arg.(int64)))
			argc += 2
			break
		case float32:
			if paramTypes[i] != C.WASM_F32 {
				str := "CallFunc error: invalid param type %d, " +
					"expect f32 but got other"
				return fmt.Errorf(str, paramTypes[i])
			}
			*(*C.float)(unsafe.Pointer(&argv[argc])) = (C.float)(arg.(float32))
			argc++
			break
		case float64:
			if paramTypes[i] != C.WASM_F64 {
				str := "CallFunc error: invalid param type %d, " +
					"expect f64 but got other"
				return fmt.Errorf(str, paramTypes[i])
			}
			addr := (*C.uint32_t)(unsafe.Pointer(&argv[argc]))
			C.PUT_F64_TO_ADDR(addr, (C.double)(arg.(float64)))
			argc += 2
			break
		default:
			return fmt.Errorf("CallFunc error: unknown param type %d",
				paramTypes[i])
		}
		i++
	}

	if i < paramCount {
		str := "CallFunc error: invalid param count, " +
			"must be no smaller than %d"
		return fmt.Errorf(str, paramCount)
	}
	err := inst.CallFunc(funcName, argc, argv)
	if err != nil {
		return err
	}

	argc = 0
	for i = 0; i < resultCount; i++ {

		switch resultTypes[i] {
		case C.WASM_I32, C.WASM_FUNCREF, C.WASM_ANYREF:
			i32 := (int32)(argv[argc])
			results[i] = i32
			argc++
			break
		case C.WASM_I64:
			addr := (*C.uint32_t)(unsafe.Pointer(&argv[argc]))
			results[i] = (int64)(C.GET_I64_FROM_ADDR(addr))
			argc += 2
			break
		case C.WASM_F32:
			addr := (*C.float)(unsafe.Pointer(&argv[argc]))
			results[i] = (float32)(*addr)
			argc++
			break
		case C.WASM_F64:
			addr := (*C.uint32_t)(unsafe.Pointer(&argv[argc]))
			results[i] = (float64)(C.GET_F64_FROM_ADDR(addr))
			argc += 2
			break

		default:
			break
		}
	}

	return nil
}

// GetException Get exception info of the instance
func (inst *Instance) GetException() string {
	cStr := C.wasm_runtime_get_exception(inst._instance)
	goStr := C.GoString(cStr)
	return goStr
}

// ModuleMalloc Allocate memory from the heap of the instance
func (inst Instance) ModuleMalloc(size uint32) (uint32, *uint8) {
	var offset C.uint32_t
	nativeAddrs := make([]*uint8, 1, 1)
	ptr := unsafe.Pointer(&nativeAddrs[0])
	offset = C.wasm_runtime_module_malloc(inst._instance, (C.uint32_t)(size),
		(*unsafe.Pointer)(ptr))
	return (uint32)(offset), nativeAddrs[0]
}

// ModuleFree Free memory to the heap of the instance */
func (inst Instance) ModuleFree(offset uint32) {
	C.wasm_runtime_module_free(inst._instance, (C.uint32_t)(offset))
}

// ValidateAppAddr ValidateAppAddr
func (inst Instance) ValidateAppAddr(appOffset uint32, size uint32) bool {
	ret := C.wasm_runtime_validate_app_addr(inst._instance,
		(C.uint32_t)(appOffset),
		(C.uint32_t)(size))
	return (bool)(ret)
}

// ValidateStrAddr ValidateStrAddr
func (inst Instance) ValidateStrAddr(appStrOffset uint32) bool {
	ret := C.wasm_runtime_validate_app_str_addr(inst._instance,
		(C.uint32_t)(appStrOffset))
	return (bool)(ret)
}

// ValidateNativeAddr ValidateNativeAddr
func (inst Instance) ValidateNativeAddr(nativePtr *uint8, size uint32) bool {
	nativePtrC := (unsafe.Pointer)(nativePtr)
	ret := C.wasm_runtime_validate_native_addr(inst._instance,
		nativePtrC,
		(C.uint32_t)(size))
	return (bool)(ret)
}

// AddrAppToNative AddrAppToNative
func (inst Instance) AddrAppToNative(appOffset uint32) *uint8 {
	nativePtr := C.wasm_runtime_addr_app_to_native(inst._instance,
		(C.uint32_t)(appOffset))
	return (*uint8)(nativePtr)
}

// AddrNativeToApp AddrNativeToApp
func (inst Instance) AddrNativeToApp(nativePtr *uint8) uint32 {
	nativePtrC := (unsafe.Pointer)(nativePtr)
	offset := C.wasm_runtime_addr_native_to_app(inst._instance,
		nativePtrC)
	return (uint32)(offset)
}

// GetAppAddrRange GetAppAddrRange
func (inst Instance) GetAppAddrRange(appOffset uint32) (bool,
	uint32,
	uint32) {
	var startOffset, endOffset C.uint32_t
	ret := C.wasm_runtime_get_app_addr_range(inst._instance,
		(C.uint32_t)(appOffset),
		&startOffset, &endOffset)
	return (bool)(ret), (uint32)(startOffset), (uint32)(endOffset)
}

// GetNativeAddrRange GetNativeAddrRange
func (inst Instance) GetNativeAddrRange(nativePtr *uint8) (bool,
	*uint8,
	*uint8) {
	var startAddr, endAddr *C.uint8_t
	nativePtrC := (*C.uint8_t)((unsafe.Pointer)(nativePtr))
	ret := C.wasm_runtime_get_native_addr_range(inst._instance,
		nativePtrC,
		&startAddr, &endAddr)
	return (bool)(ret), (*uint8)(startAddr), (*uint8)(endAddr)
}

// DumpMemoryConsumption DumpMemoryConsumption
func (inst Instance) DumpMemoryConsumption() {
	C.wasm_runtime_dump_mem_consumption(inst._execEnv)
}

// DumpCallStack DumpCallStack
func (inst Instance) DumpCallStack() {
	C.wasm_runtime_dump_call_stack(inst._execEnv)
}
