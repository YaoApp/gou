# Yao WASM Runtime — Multi-Language Script Support

## Overview

Yao currently executes user scripts (Hooks, Tools, Guards, etc.) through the V8 JavaScript engine. We are adding **WASM as a parallel runtime** to enable **multi-language support** — allowing developers to write Yao scripts in **Rust, Go, C**, or any language that compiles to WebAssembly. Existing **TypeScript/JavaScript** scripts can also be compiled to WASM via `yao build` with **zero code changes**.

```
gou/runtime/
├── v8/         ← Existing: TypeScript/JavaScript (unchanged)
├── wasm/       ← New: WebAssembly (Rust, Go, C, TS→WASM...)
└── transform/  ← Existing: TS/JS compilation (unchanged)
```

## Motivation

1. **Multi-Language Support**: Write Yao scripts in Rust, Go, C, or continue with TypeScript.

2. **Binary Distribution**: WASM modules are compiled binaries. Compiled-language WASM (Rust, Go, C) is highly resistant to reverse engineering — similar to native binaries. TS/JS compiled via QuickJS retains string constants and function names in bytecode, offering limited source protection. For sensitive business logic, implement critical algorithms in Rust or Go.

3. **Lightweight Runtime**: [wazero](https://wazero.io/) is a zero-dependency WebAssembly runtime in pure Go. No V8/C++ dependency for script execution.

4. **Edge & Embedded Deployment**: A WASM-only Yao Runtime for IoT, edge, and serverless.

## Architecture

### Two Products, One Codebase

**Yao** (Development & Full-Featured):
- V8 + WASM dual runtime
- Supports `.ts`, `.js`, `.wasm` scripts
- `yao build` compiles TS/JS → WASM for deployment
- When both `hook.ts` and `hook.wasm` exist, **WASM takes priority**
- Full development toolchain: hot reload, debugger, REPL

**Yao Runtime** (Production & Deployment):
- WASM-only — no V8, no C++ dependency for script execution
- Runs `.wasm` modules exclusively
- Minimal footprint, cross-compiles to any platform
- Applications ship as `.wasm` binaries — no source code distribution

```
  Developer Machine (Yao)              Production (Yao Runtime)
  ┌─────────────────────┐              ┌─────────────────────┐
  │  Write TS/JS/Rust/Go │              │                     │
  │         ↓            │   yao build  │  Load .wasm only    │
  │  Run & Debug with V8 │  ─────────→  │  No V8 dependency   │
  │  + WASM dual runtime │              │  ~500KB per request  │
  │         ↓            │              │  Instant GC          │
  │  yao build → .wasm   │              │                     │
  └─────────────────────┘              └─────────────────────┘
```

### Unified Process System

Callers don't know or care what language a script is written in. Everything goes through Yao's existing Process system:

```
Process("scripts.validate.Check", args)     // Could be TS, Rust, Go, or C
Process("scripts.transform.Convert", args)  // Caller doesn't know, doesn't care
```

### Yao Host API (ABI)

WASM modules communicate with Yao through **Host Functions** — standard WASM imports provided by the Go host:

| Function | Signature | Description |
|----------|-----------|-------------|
| `process` | `(name_ptr, name_len, args_ptr, args_len) → packed_result` | Call any Yao Process |
| `log` | `(level, msg_ptr, msg_len)` | Write to Yao log |

Since `process` is the gateway to Yao's entire capability set (Models, Flows, Tables, APIs, Stores, Queries, etc.), a single host function covers the majority of use cases.

### How Each Language Uses It

**Rust** (native WASM, direct host function import):
```rust
#[link(wasm_import_module = "yao")]
extern "C" {
    fn process(name_ptr: *const u8, name_len: u32,
               args_ptr: *const u8, args_len: u32) -> u64;
}

#[no_mangle]
pub extern "C" fn before_save(payload_ptr: *const u8, payload_len: u32) -> u64 {
    let user = yao_process("models.user.Find", &[user_id]);
    // ... business logic ...
}
```

**Go** (native WASM via TinyGo / wasip1):
```go
//go:wasmimport yao process
func yaoProcess(namePtr, nameLen, argsPtr, argsLen uint32) uint64

//export BeforeSave
func BeforeSave() {
    user := YaoProcess("models.user.Find", []any{userID})
    // ... business logic ...
}
```

**C** (native WASM via clang):
```c
__attribute__((import_module("yao"), import_name("process")))
extern uint64_t yao_process(uint32_t name_ptr, uint32_t name_len,
                              uint32_t args_ptr, uint32_t args_len);

__attribute__((export_name("before_save")))
uint64_t before_save(uint32_t payload_ptr, uint32_t payload_len) {
    // ... call yao_process ...
}
```

**TypeScript** (zero code changes — compiled to WASM via `yao build`):
```typescript
// This is existing Yao TS code. No modifications needed.
// `yao build` compiles it to WASM automatically.
function BeforeSave(payload: Record<string, any>): Record<string, any> {
    const user = Process("models.user.Find", payload.user_id, {});
    const fs = new FS("/data/app");
    const config = fs.ReadFile("config.json");
    if (payload.amount > 10000) {
        payload.status = "pending_approval";
        payload.reviewer = user.name;
    }
    return payload;
}
```

### TS/JS → WASM Compilation

TypeScript/JavaScript cannot be directly AOT-compiled to native WASM (it's a dynamic language). The `yao build` pipeline uses a custom [Javy](https://github.com/bytecodealliance/javy) plugin built with the [QuickJS-NG](https://github.com/quickjs-ng/quickjs) engine:

```
hook.ts → esbuild → hook.js → Javy (Yao Plugin) → hook.wasm (1.5KB)
```

The **Yao Plugin** (1.2MB, loaded once at startup) is a custom QuickJS WASM module that pre-injects all Yao Runtime APIs (`Process`, `log`, `FS`, `Store`, `Http`, etc.) as global JavaScript objects. These APIs bridge to Go host functions via WASM imports. The user's TS/JS code requires **zero modifications**.

Rust, Go, and C compile to **native WASM** — no interpreter, no QuickJS, direct host function calls. Native WASM is also highly resistant to reverse engineering, making it suitable for proprietary algorithms and sensitive business logic.

## Proof of Concept Results

All tests conducted with verified, working code (`wasm-poc/`).

### End-to-End Host Function Bridge (Verified)

TS code calling `Process()` → Go host receives call + args → Go returns result → TS receives structured data:

```
[Host] log.info: === BeforeSave Start ===
[Host] Process("fs.ReadFile", [/data/app/config.json])           ← new FS("/data/app").ReadFile()
[Host] Process("models.user.Find", [100 map[]])                  ← Process() call
[Host] Process("store.Get", [voucher:42])                        ← new Store("voucher:").Get()
[Host] Process("fs.WriteFile", [/data/app/approval/42.json ...]) ← fs.WriteFile()
[Host] Process("http.Post", [https://api.example.com/notify ...])← new Http(...).Post()
[Host] Process("store.Set", [voucher:42 map[...]])               ← cache.Set()
[Host] Process("fs.Exists", [/data/app/approval/42.json])        ← fs.Exists()
[Host] Process("fs.ReadDir", [/data/app/approval])               ← fs.ReadDir()
[Host] log.info: === BeforeSave Done ===
{"id":42,"user_id":100,"amount":15000,"status":"pending_approval","reviewer":"张三"}
```

Verified capabilities:
- `Process(name, ...args)` with structured data round-trip ✅
- `new FS(basePath)` constructor + `.ReadFile()`, `.WriteFile()`, `.Exists()`, `.ReadDir()` ✅
- `new Store(prefix)` constructor + `.Get()`, `.Set()` ✅
- `new Http(baseURL)` constructor + `.Post()` ✅
- `log.Info()`, `log.Warn()` ✅
- JSON serialization/deserialization across WASM boundary ✅
- Object property access on returned data (`user.name`) ✅
- **Zero TS code modifications** ✅

### Script Sizes

| Script | Size |
|--------|------|
| Simple Hook (TS→WASM) | **1.5 KB** |
| Complex Hook with FS/Store/Http (TS→WASM) | **3.2 KB** |
| Yao Plugin (QuickJS engine, shared, loaded once) | **1.2 MB** |
| **50 TS scripts total** | **~1.3 MB** |

### Performance

Measured with Go-side precise timing (not JS `Date.now()`):

**Pure computation benchmark** (fibonacci recursive 35, 100K string concat, 50K JSON ops, 100K array sort):

| Engine | Total Time |
|--------|-----------|
| V8 (Node.js) | **177ms** |
| QuickJS (WASM via wazero) | **4,296ms** (~24x slower) |

**Why this doesn't matter for Yao**: Real Yao Hooks spend 90%+ of execution time in `Process()` calls (database queries, network I/O — typically 10-500ms each). The JS logic itself is simple conditionals and field assignments. A Hook that takes 200ms with V8 would take ~202ms with QuickJS — the difference is imperceptible.

**Startup & instantiation:**

| Step | Time |
|------|------|
| Plugin instantiate (once at startup) | **220µs** |
| Script compile (once per script) | **220µs** |
| Per-request execute (instantiate + run + dispose) | **~2ms** |

### Memory

| Mode | Per-Request Memory | Reclamation |
|------|-------------------|-------------|
| V8 (Standard mode) | ~50-100MB (Isolate) | Lazy on macOS (`MADV_FREE`) |
| WASM (wazero) | **~500KB** | **Immediate** |

## Proposed Directory Structure

```
gou/runtime/
├── v8/                          ← Existing (unchanged)
│   ├── bridge/
│   ├── functions/
│   ├── objects/
│   └── ...
├── wasm/                        ← New
│   ├── runtime.go               ← wazero runtime lifecycle
│   ├── script.go                ← Load, Compile, Exec WASM scripts
│   ├── process.go               ← process.Register for WASM scripts
│   ├── plugin/                  ← Yao Plugin (Rust/QuickJS)
│   │   ├── src/lib.rs           ← Plugin source (injects Process, FS, Store, Http, log)
│   │   └── Cargo.toml
│   ├── host/                    ← Host function implementations
│   │   ├── process.go           ← yao.process host function
│   │   └── log.go               ← yao.log host function
│   └── bridge/                  ← Go ↔ WASM memory serialization
│       ├── encode.go
│       └── decode.go
└── transform/                   ← Existing (unchanged)
```

## Roadmap

Status: **Planned**

| Release | Scope |
|---------|-------|
| **v1.1-alpha** | Core WASM runtime + `process` host function + full Yao API injection |
| **v1.1-alpha** | `yao build` command: TS → JS → WASM pipeline |
| **v1.2-beta** | **Yao Runtime** standalone binary (WASM-only, no V8) |
| **v1.3-release** | Language SDKs (Rust crate, Go package, etc.) |

*Versions and timeline are subject to adjustment based on development progress.*

## References

- [WebAssembly Specification](https://webassembly.org/)
- [wazero — Zero-dependency Go WebAssembly Runtime](https://wazero.io/)
- [Javy — JavaScript to WebAssembly Toolchain](https://github.com/bytecodealliance/javy)
- [QuickJS-NG — QuickJS, the Next Generation](https://github.com/quickjs-ng/quickjs)
- [WASI — WebAssembly System Interface](https://wasi.dev/)
