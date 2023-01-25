/*
 * Copyright (C) 2019 Intel Corporation.  All rights reserved.
 * SPDX-License-Identifier: Apache-2.0 WITH LLVM-exception
 */

package wamr

/*
#include <stdlib.h>
#include <string.h>

#include <wasm_export.h>

void
bh_log_set_verbose_level(uint32_t level);

bool
init_wamr_runtime(bool alloc_with_pool, uint8_t *heap_buf,
                  uint32_t heap_size, uint32_t maxThreadNum)
{
    RuntimeInitArgs init_args;

    memset(&init_args, 0, sizeof(RuntimeInitArgs));

    if (alloc_with_pool) {
        init_args.mem_alloc_type = alloc_with_pool;
        init_args.mem_alloc_option.pool.heap_buf = heap_buf;
        init_args.mem_alloc_option.pool.heap_size = heap_size;
    }
    else {
        init_args.mem_alloc_type = Alloc_With_System_Allocator;
    }

    return wasm_runtime_full_init(&init_args);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// LogLevel alias uint32
type LogLevel uint32

const (
	// LogLevelFatal Fatal
	LogLevelFatal LogLevel = 0
	// LogLevelError error
	LogLevelError LogLevel = 1
	// LogLevelWarning warning
	LogLevelWarning LogLevel = 2
	// LogLevelDebug debug
	LogLevelDebug LogLevel = 3
	// LogLevelVerbose verbose
	LogLevelVerbose LogLevel = 4
)

/*
	type NativeSymbol struct {
	    symbol string
	    func_ptr *uint8
	    signature string
	}
*/

// RuntimeSingleton struct
type RuntimeSingleton struct {
	initialized bool
}

var _runtimeSingleton *RuntimeSingleton

// Runtime Return the runtime singleton
func Runtime() *RuntimeSingleton {
	if _runtimeSingleton == nil {
		runtime := &RuntimeSingleton{}
		_runtimeSingleton = runtime
	}
	return _runtimeSingleton
}

// FullInit Initialize the WASM runtime environment
func (runtime *RuntimeSingleton) FullInit(allocWithPool bool, heapBuf []byte,
	maxThreadNum uint) error {
	var heapBufC *C.uchar

	if runtime.initialized {
		return nil
	}

	if allocWithPool {
		if heapBuf == nil {
			return fmt.Errorf("Failed to init WAMR runtime")
		}
		heapBufC = (*C.uchar)(unsafe.Pointer(&heapBuf[0]))
	}

	if !C.init_wamr_runtime((C.bool)(allocWithPool), heapBufC,
		(C.uint)(len(heapBuf)),
		(C.uint)(maxThreadNum)) {
		return fmt.Errorf("Failed to init WAMR runtime")
	}

	runtime.initialized = true
	return nil
}

// Init Initialize the WASM runtime environment
func (runtime *RuntimeSingleton) Init() error {
	return runtime.FullInit(false, nil, 1)
}

// Destroy the WASM runtime environment
func (runtime *RuntimeSingleton) Destroy() {
	if runtime.initialized {
		C.wasm_runtime_destroy()
		runtime.initialized = false
	}
}

// SetLogLevel Set log verbose level (0 to 5, default is 2), 	larger level with more log
func (runtime *RuntimeSingleton) SetLogLevel(level LogLevel) {
	C.bh_log_set_verbose_level(C.uint32_t(level))
}

/*
func (runtime *RuntimeSingleton) RegisterNatives(moduleName string,
                                      nativeSymbols []NativeSymbol) {
}
*/ /* TODO */

// InitThreadEnv InitThreadEnv
func (runtime *RuntimeSingleton) InitThreadEnv() bool {
	if !C.wasm_runtime_init_thread_env() {
		return false
	}
	return true
}

// DestroyThreadEnv DestroyThreadEnv
func (runtime *RuntimeSingleton) DestroyThreadEnv() {
	C.wasm_runtime_destroy_thread_env()
}

// ThreadEnvInited ThreadEnvInited
func (runtime *RuntimeSingleton) ThreadEnvInited() bool {
	if !C.wasm_runtime_thread_env_inited() {
		return false
	}
	return true
}

// Malloc Allocate memory from runtime memory environment
func (runtime *RuntimeSingleton) Malloc(size uint32) *uint8 {
	ptr := C.wasm_runtime_malloc((C.uint32_t)(size))
	return (*uint8)(ptr)
}

// Free Free memory to runtime memory environment
func (runtime *RuntimeSingleton) Free(ptr *uint8) {
	C.wasm_runtime_free((unsafe.Pointer)(ptr))
}
