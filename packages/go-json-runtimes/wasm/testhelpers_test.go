package wasm

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// buildMinimalWasmWithMalloc creates a valid WASM module using wazero's host module builder
// then extracts the compiled bytes. Since wazero doesn't expose WAT compilation,
// we build a module programmatically and verify it compiles.
func buildMinimalWasmWithMalloc(t *testing.T) []byte {
	t.Helper()
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// Build a host module that exports malloc and free, then get its compiled form.
	// Actually, host modules can't be exported as WASM bytes.
	// Instead, use a known-good minimal WASM binary.
	// This binary was generated from:
	//   (module
	//     (memory (export "memory") 1)
	//     (func $malloc (export "malloc") (param i32) (result i32)
	//       i32.const 1024)
	//     (func $free (export "free") (param i32 i32))
	//   )
	return validMinimalWasm()
}

// buildWasmWithoutMalloc creates a valid WASM module that exports a function but NOT malloc.
func buildWasmWithoutMalloc(t *testing.T) []byte {
	t.Helper()
	// (module
	//   (memory (export "memory") 1)
	//   (func $test (export "test") (param i32 i32) (result i64)
	//     i64.const 0)
	// )
	return validNoMallocWasm()
}

// buildInfiniteLoopWasm creates a WASM module with malloc and a function that loops forever.
func buildInfiniteLoopWasm(t *testing.T) []byte {
	t.Helper()
	// (module
	//   (memory (export "memory") 1)
	//   (func $malloc (export "malloc") (param i32) (result i32)
	//     i32.const 1024)
	//   (func $loop (export "loop") (param i32 i32) (result i64)
	//     (loop $l (br $l))
	//     i64.const 0)
	// )
	return validInfiniteLoopWasm()
}

// buildEchoWasm returns nil — full protocol tests require pre-compiled WASM from Rust/TinyGo.
func buildEchoWasm(t *testing.T) []byte {
	t.Helper()
	return nil
}

// validMinimalWasm returns a valid WASM binary with memory + malloc + free.
// Generated and verified against wazero compilation.
func validMinimalWasm() []byte {
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// Use host module approach: create a module with Go functions that act as malloc/free
	// This won't work for our test since we need a WASM module (not host module).
	// Instead, let's verify our binary is correct by building it properly.

	// Correct WASM binary for:
	// (module
	//   (type (;0;) (func (param i32) (result i32)))
	//   (type (;1;) (func (param i32 i32)))
	//   (func (;0;) (type 0) (param i32) (result i32) i32.const 1024)
	//   (func (;1;) (type 1) (param i32 i32))
	//   (memory (;0;) 1)
	//   (export "memory" (memory 0))
	//   (export "malloc" (func 0))
	//   (export "free" (func 1))
	// )
	return encodeModule(
		// types
		[]funcType{
			{params: []byte{i32}, results: []byte{i32}},       // malloc
			{params: []byte{i32, i32}, results: []byte{}},     // free
		},
		// functions (type indices)
		[]uint32{0, 1},
		// memory: 1 page min
		1,
		// exports
		[]export{
			{name: "memory", kind: 0x02, index: 0},
			{name: "malloc", kind: 0x00, index: 0},
			{name: "free", kind: 0x00, index: 1},
		},
		// code
		[][]byte{
			{0x00, 0x41, 0x80, 0x08, 0x0b},       // malloc: (local) i32.const 1024, end
			{0x00, 0x0b},                           // free: (local) end
		},
	)
}

func validNoMallocWasm() []byte {
	return encodeModule(
		[]funcType{
			{params: []byte{i32, i32}, results: []byte{i64}},
		},
		[]uint32{0},
		1,
		[]export{
			{name: "memory", kind: 0x02, index: 0},
			{name: "test", kind: 0x00, index: 0},
		},
		[][]byte{
			{0x00, 0x42, 0x00, 0x0b}, // (local) i64.const 0, end
		},
	)
}

func validInfiniteLoopWasm() []byte {
	return encodeModule(
		[]funcType{
			{params: []byte{i32}, results: []byte{i32}},       // malloc
			{params: []byte{i32, i32}, results: []byte{i64}},  // loop
		},
		[]uint32{0, 1},
		1,
		[]export{
			{name: "memory", kind: 0x02, index: 0},
			{name: "malloc", kind: 0x00, index: 0},
			{name: "loop", kind: 0x00, index: 1},
		},
		[][]byte{
			{0x00, 0x41, 0x80, 0x08, 0x0b},                         // malloc: i32.const 1024, end
			{0x00, 0x03, 0x40, 0x0c, 0x00, 0x0b, 0x42, 0x00, 0x0b}, // loop: loop{br 0}end, i64.const 0, end
		},
	)
}

// --- WASM binary encoding helpers ---

const (
	i32 byte = 0x7f
	i64 byte = 0x7e
)

type funcType struct {
	params  []byte
	results []byte
}

type export struct {
	name  string
	kind  byte
	index uint32
}

func encodeModule(types []funcType, funcIndices []uint32, memPages uint32, exports []export, codes [][]byte) []byte {
	var buf []byte
	buf = append(buf, 0x00, 0x61, 0x73, 0x6d) // magic
	buf = append(buf, 0x01, 0x00, 0x00, 0x00) // version

	// Type section (id=1)
	typeSection := encodeTypeSection(types)
	buf = append(buf, 0x01)
	buf = append(buf, encodeU32(uint32(len(typeSection)))...)
	buf = append(buf, typeSection...)

	// Function section (id=3)
	funcSection := encodeFuncSection(funcIndices)
	buf = append(buf, 0x03)
	buf = append(buf, encodeU32(uint32(len(funcSection)))...)
	buf = append(buf, funcSection...)

	// Memory section (id=5)
	memSection := encodeMemSection(memPages)
	buf = append(buf, 0x05)
	buf = append(buf, encodeU32(uint32(len(memSection)))...)
	buf = append(buf, memSection...)

	// Export section (id=7)
	expSection := encodeExportSection(exports)
	buf = append(buf, 0x07)
	buf = append(buf, encodeU32(uint32(len(expSection)))...)
	buf = append(buf, expSection...)

	// Code section (id=10)
	codeSection := encodeCodeSection(codes)
	buf = append(buf, 0x0a)
	buf = append(buf, encodeU32(uint32(len(codeSection)))...)
	buf = append(buf, codeSection...)

	return buf
}

func encodeTypeSection(types []funcType) []byte {
	var buf []byte
	buf = append(buf, encodeU32(uint32(len(types)))...)
	for _, t := range types {
		buf = append(buf, 0x60) // func type marker
		buf = append(buf, encodeU32(uint32(len(t.params)))...)
		buf = append(buf, t.params...)
		buf = append(buf, encodeU32(uint32(len(t.results)))...)
		buf = append(buf, t.results...)
	}
	return buf
}

func encodeFuncSection(indices []uint32) []byte {
	var buf []byte
	buf = append(buf, encodeU32(uint32(len(indices)))...)
	for _, idx := range indices {
		buf = append(buf, encodeU32(idx)...)
	}
	return buf
}

func encodeMemSection(pages uint32) []byte {
	var buf []byte
	buf = append(buf, encodeU32(1)...) // 1 memory
	buf = append(buf, 0x00)            // no max
	buf = append(buf, encodeU32(pages)...)
	return buf
}

func encodeExportSection(exports []export) []byte {
	var buf []byte
	buf = append(buf, encodeU32(uint32(len(exports)))...)
	for _, e := range exports {
		buf = append(buf, encodeU32(uint32(len(e.name)))...)
		buf = append(buf, []byte(e.name)...)
		buf = append(buf, e.kind)
		buf = append(buf, encodeU32(e.index)...)
	}
	return buf
}

func encodeCodeSection(codes [][]byte) []byte {
	var buf []byte
	buf = append(buf, encodeU32(uint32(len(codes)))...)
	for _, code := range codes {
		buf = append(buf, encodeU32(uint32(len(code)))...)
		buf = append(buf, code...)
	}
	return buf
}

func encodeU32(v uint32) []byte {
	var buf []byte
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		buf = append(buf, b)
		if v == 0 {
			break
		}
	}
	return buf
}

// verifyWasmCompiles checks that a WASM binary is valid by compiling it with wazero.
func verifyWasmCompiles(t *testing.T, wasmBytes []byte) {
	t.Helper()
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		t.Fatalf("WASM binary failed to compile: %v", err)
	}
	compiled.Close(ctx)
}

// verifyWasmExports checks that a compiled module has the expected exports.
func verifyWasmExports(t *testing.T, wasmBytes []byte, expectedExports []string) {
	t.Helper()
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	defer compiled.Close(ctx)

	defs := compiled.ExportedFunctions()
	for _, name := range expectedExports {
		if _, ok := defs[name]; !ok {
			if name == "memory" {
				continue // memory is not in ExportedFunctions
			}
			t.Errorf("expected export '%s' not found", name)
		}
	}
}

// Ensure test modules compile correctly
func TestTestModules_Compile(t *testing.T) {
	t.Run("MinimalWithMalloc", func(t *testing.T) {
		wasm := validMinimalWasm()
		verifyWasmCompiles(t, wasm)
		verifyWasmExports(t, wasm, []string{"malloc", "free"})
	})

	t.Run("NoMalloc", func(t *testing.T) {
		wasm := validNoMallocWasm()
		verifyWasmCompiles(t, wasm)
		verifyWasmExports(t, wasm, []string{"test"})
	})

	t.Run("InfiniteLoop", func(t *testing.T) {
		wasm := validInfiniteLoopWasm()
		verifyWasmCompiles(t, wasm)
		verifyWasmExports(t, wasm, []string{"malloc", "loop"})
	})
}

// TestWasmRuntime_ExecuteWithMalloc verifies the full Execute flow with a module
// that has malloc but the called function returns 0 (empty result).
func TestWasmRuntime_ExecuteWithMalloc(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	ctx := context.Background()

	// Build a module with malloc + a function that returns i64(0) = empty result
	wasmBytes := encodeModule(
		[]funcType{
			{params: []byte{i32}, results: []byte{i32}},       // malloc
			{params: []byte{i32, i32}, results: []byte{}},     // free
			{params: []byte{i32, i32}, results: []byte{i64}},  // process
		},
		[]uint32{0, 1, 2},
		1,
		[]export{
			{name: "memory", kind: 0x02, index: 0},
			{name: "malloc", kind: 0x00, index: 0},
			{name: "free", kind: 0x00, index: 1},
			{name: "process", kind: 0x00, index: 2},
		},
		[][]byte{
			{0x00, 0x41, 0x80, 0x08, 0x0b},  // malloc: i32.const 1024, end
			{0x00, 0x0b},                      // free: end
			{0x00, 0x42, 0x00, 0x0b},          // process: i64.const 0, end (returns empty)
		},
	)

	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "process.wasm")
	os.WriteFile(wasmFile, wasmBytes, 0644)

	result, err := rt.Execute(ctx, wasmFile, "process", map[string]any{"test": true}, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	// Function returns i64(0) which means resultLen=0, so result should be nil
	if result != nil {
		t.Errorf("expected nil result for empty return, got %v", result)
	}
}

// Ensure the unused import is used
var _ api.Module
