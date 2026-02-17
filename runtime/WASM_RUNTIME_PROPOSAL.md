# Yao WASM Runtime — Multi-Language Script Support

## Overview

Yao currently executes user scripts (Hooks, Tools, Guards, etc.) through the V8 JavaScript engine, which works well for TypeScript/JavaScript development. We are exploring **adding WASM as a parallel runtime** to enable **multi-language support** — allowing developers to write Yao scripts in **Rust, Go, C, AssemblyScript**, or any language that compiles to WebAssembly, alongside existing TypeScript/JavaScript.

This is **not a replacement** for V8. It's a new runtime that sits alongside V8 under the existing `runtime/` architecture:

```
gou/runtime/
├── v8/         ← Existing: TypeScript/JavaScript (unchanged)
├── wasm/       ← Proposed: WebAssembly (Rust, Go, C, AssemblyScript, TS→WASM...)
└── transform/  ← Existing: TS/JS compilation (unchanged)
```

## Motivation

1. **Multi-Language Support**: Let developers write Yao scripts in languages they're most productive in — Rust for performance-critical hooks, Go for system-level operations, or continue with TypeScript for rapid prototyping.

2. **Language-Agnostic Binary Format**: WASM is a W3C standard supported by all major languages. Once Yao defines its Host API, any language can target it.

3. **Binary Distribution**: WASM modules are compiled binaries, enabling applications to be distributed without source code — useful for commercial Yao applications.

4. **Lightweight Runtime**: [wazero](https://wazero.io/) is a zero-dependency WebAssembly runtime written in pure Go. The WASM layer itself requires no CGO, eliminating the V8/C++ dependency from the script execution path.

5. **Edge & Embedded Deployment**: A WASM-only Yao Runtime could run on IoT devices, edge nodes, or serverless platforms where V8 is too heavy.

## Design Principles

### Unified Process System

The core design principle: **callers don't know or care what language a script is written in.** Everything goes through Yao's existing Process system:

```
Process("scripts.validate.Check", args)     // Could be TS, Rust, Go, or C
Process("scripts.transform.Convert", args)  // Caller doesn't know, doesn't care
```

WASM scripts register into the Process system the same way V8 scripts do today:

```go
// V8 (existing)
process.Register("scripts", processScriptsV8)

// WASM (proposed)
process.Register("scripts", processScriptsWASM)

// Or both, with file extension routing:
// .ts/.js → V8, .wasm → wazero
```

### Yao Host API (ABI)

WASM modules communicate with Yao through **Host Functions** — standard WASM imports provided by the Go host. This is the **Yao WASM ABI**:

```wasm
;; Every Yao WASM module imports from the "yao" namespace
(import "yao" "process" (func (param i32 i32 i32 i32) (result i64)))
(import "yao" "log"     (func (param i32 i32 i32)))
```

The host function signatures:

| Function | Signature | Description |
|----------|-----------|-------------|
| `process` | `(name_ptr, name_len, args_ptr, args_len) → packed_result` | Call any Yao Process |
| `log` | `(level, msg_ptr, msg_len)` | Write to Yao log |

Since `process` is the gateway to Yao's entire capability set (Models, Flows, Tables, APIs, Stores, Queries, etc.), **a single host function covers 80%+ of use cases**. Additional host functions (store, query, fs, http) can be added for performance-critical paths that want to avoid the Process dispatch overhead.

### How Each Language Uses It

**Rust:**
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

**Go (TinyGo / Go 1.21+ wasip1):**
```go
//go:wasmimport yao process
func yaoProcess(namePtr, nameLen, argsPtr, argsLen uint32) uint64

//export BeforeSave
func BeforeSave() {
    user := YaoProcess("models.user.Find", []any{userID})
    // ... business logic ...
}
```

**AssemblyScript (TS-like syntax, compiles to WASM natively):**
```typescript
@external("yao", "process")
declare function yao_process(namePtr: u32, nameLen: u32,
                              argsPtr: u32, argsLen: u32): u64;

export function beforeSave(payloadPtr: u32, payloadLen: u32): u64 {
    const user = Process("models.user.Find", [userId]);
    // ... business logic ...
}
```

**C:**
```c
__attribute__((import_module("yao"), import_name("process")))
extern uint64_t yao_process(uint32_t name_ptr, uint32_t name_len,
                              uint32_t args_ptr, uint32_t args_len);

__attribute__((export_name("before_save")))
uint64_t before_save(uint32_t payload_ptr, uint32_t payload_len) {
    // ... call yao_process ...
}
```

**TypeScript (via Javy/QuickJS → WASM compilation):**
```typescript
// Existing TS code compiles to WASM via: esbuild → JS → Javy → .wasm
// No code changes needed — the build toolchain handles it
function BeforeSave(payload: any): any {
    const user = Process("models.user.Find", payload.user_id);
    return { ...payload, user_name: user.name };
}
```

### Two Products, One Codebase

The WASM runtime enables two distinct distribution artifacts from the same codebase:

**Yao** (Development & Full-Featured):
- V8 + WASM dual runtime
- Supports `.ts`, `.js`, `.wasm` scripts
- `yao build` compiles TS/JS → WASM for deployment
- When both `hook.ts` and `hook.wasm` exist, **WASM takes priority**
- Full development toolchain: hot reload, debugger, REPL

**Yao Runtime** (Production & Deployment):
- WASM-only — no V8, no C++ dependency for script execution
- Runs `.wasm` modules exclusively
- Minimal binary size, cross-compiles to any platform
- Suitable for edge, IoT, serverless, and embedded deployment
- Applications ship as `.wasm` binaries — no source code distribution

```
Development workflow:

  Developer Machine (Yao)              Production (Yao Runtime)
  ┌─────────────────────┐              ┌─────────────────────┐
  │  Write TS/JS/Rust/Go │              │                     │
  │         ↓            │   yao build  │  Load .wasm only    │
  │  Run & Debug with V8 │  ─────────→  │  Pure Go binary     │
  │  + WASM dual runtime │              │  No V8 dependency    │
  │         ↓            │              │  ~500KB per request  │
  │  yao build → .wasm   │              │                     │
  └─────────────────────┘              └─────────────────────┘
```

### Script Lifecycle

**Yao (dual runtime):**
```
Yao Start
├── Load V8 engine
├── Load WASM runtime (wazero) + QuickJS Plugin (~340ms, once)
├── Scan scripts/ directory
│   ├── validate.ts              → V8
│   ├── hook.ts + hook.wasm      → WASM (priority)
│   ├── transform.wasm (Rust)    → WASM
│   └── plugin.wasm (Go)         → WASM
└── Ready to serve (V8 + WASM)
```

**Yao Runtime (WASM only):**
```
Yao Runtime Start
├── Load WASM runtime (wazero) + QuickJS Plugin (~340ms, once)
├── Scan scripts/ directory
│   ├── validate.wasm (3KB)    → Compile: ~0.02ms → Cache
│   ├── hook.wasm (3KB)        → Compile: ~0.02ms → Cache
│   ├── transform.wasm (15KB)  → Compile: ~0.5ms  → Cache
│   ├── plugin.wasm (50KB)     → Compile: ~2ms    → Cache
│   └── ... (50 scripts)       → Total: ~10ms
└── Ready to serve (WASM only, no V8)

Per-request execution:
  → Get cached CompiledModule → Instantiate → Execute → Dispose
  → ~0.3ms per invocation, ~500KB memory, instantly reclaimed
```

### Data Serialization

WASM linear memory is isolated from Go memory. Data exchange uses **MessagePack** (or JSON) encoding through the linear memory:

```
Go side                    WASM Linear Memory              WASM side
────────                   ──────────────────              ─────────
map[string]any   →  encode → [bytes at ptr]  →  decode →  native struct
                    write to memory            read from memory

return value     ←  decode ← [bytes at ptr]  ←  encode ←  native struct
                    read from memory           write to memory
```

## Proof of Concept Results

We've validated the core compilation and execution chain:

| Step | Result | Performance |
|------|--------|-------------|
| TS → JS (esbuild) | ✅ Works | 1ms |
| JS → WASM (Javy, dynamic mode) | ✅ Works | ~1.6s build time |
| WASM load + execute (wazero) | ✅ Works | ~340ms to compile QuickJS plugin at startup (once), ~0.3ms per-request execute |
| Go → WASM (wasip1) | ✅ Works | Compiles and runs correctly |
| Mixed TS+Go WASM in same runtime | ✅ Works | Both execute in single wazero.Runtime |
| Host function injection | ✅ Standard WASM imports | Works across all languages |

**Script sizes (dynamic linking mode — QuickJS engine shared, not bundled per-script):**

| Script | Static Mode | Dynamic Mode |
|--------|-------------|-------------|
| Simple Hook | 1.2 MB | **3.3 KB** |
| Complex Hook | 1.2 MB | **2.8 KB** |
| QuickJS Plugin (shared, once) | — | 1.2 MB |
| **50 scripts total** | **60 MB** | **~1.35 MB** |

## Proposed Directory Structure

```
gou/runtime/
├── v8/                          ← Existing (unchanged)
│   ├── bridge/
│   ├── functions/
│   ├── objects/
│   └── ...
├── wasm/                        ← New
│   ├── runtime.go               ← wazero runtime lifecycle (Start/Stop)
│   ├── script.go                ← WasmScript: Load, Compile, Exec
│   ├── process.go               ← process.Register("scripts", ...) for WASM
│   ├── host/                    ← Host function implementations
│   │   ├── process.go           ← yao.process host function
│   │   ├── log.go               ← yao.log host function
│   │   └── ...                  ← Additional host functions as needed
│   └── bridge/                  ← Go ↔ WASM memory serialization
│       ├── encode.go
│       └── decode.go
└── transform/                   ← Existing (unchanged)
```

## Roadmap

Status: **Planned**

| Release | Phase | Scope |
|---------|-------|-------|
| **v1.1-alpha** | Core WASM runtime + `process` host function | Rust/Go/TS WASM hooks can call any Yao Process |
| **v1.1-alpha** | Full host API (log, store, query, fs, http) | Complete Yao Runtime API available in WASM |
| **v1.1-alpha** | `yao build` command: TS → JS → WASM pipeline | Existing TS apps compile to WASM with one command |
| **v1.2-beta** | **Yao Runtime** standalone binary (WASM-only, no V8) | Production deployment artifact, no V8/C++ dependency |
| **v1.3-release** | Language SDKs (Rust crate, Go package, etc.) | Ergonomic developer experience per language |

*Versions and timeline are subject to adjustment based on development progress.*

## References

- [WebAssembly Specification](https://webassembly.org/)
- [wazero — Zero-dependency Go WebAssembly Runtime](https://wazero.io/)
- [Javy — JavaScript to WebAssembly Toolchain](https://github.com/bytecodealliance/javy)
- [WASI — WebAssembly System Interface](https://wasi.dev/)
