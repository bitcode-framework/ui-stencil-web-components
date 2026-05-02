package wasm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestWasmProtocol_FixedResponse tests the full JSON-over-linear-memory protocol
// with a WASM module that returns a fixed JSON response.
//
// The module works as follows:
// 1. Host calls malloc(N) → module returns ptr 1024
// 2. Host writes JSON args to memory at ptr 1024
// 3. Host calls process(ptr=1024, len=N)
// 4. Module writes '{"value":42}' to memory at offset 2048
// 5. Module returns packed i64: (2048 << 32) | 12
// 6. Host reads 12 bytes from offset 2048 → '{"value":42}'
// 7. Host calls free(2048, 12)
// 8. Host deserializes → map[string]any{"value": 42}
func TestWasmProtocol_FixedResponse(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	wasmBytes := buildFixedResponseWasm(t)
	verifyWasmCompiles(t, wasmBytes)

	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "fixed.wasm")
	os.WriteFile(wasmFile, wasmBytes, 0644)

	result, err := rt.Execute(context.Background(), wasmFile, "process", map[string]any{"input": "test"}, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T: %v", result, result)
	}

	val, ok := resultMap["value"]
	if !ok {
		t.Fatalf("expected 'value' key in result, got: %v", resultMap)
	}

	// JSON numbers deserialize as float64
	if v, ok := val.(float64); !ok || v != 42 {
		t.Errorf("expected value=42, got %v (type %T)", val, val)
	}
}

// TestWasmProtocol_EchoInput tests that the host correctly writes args to WASM memory
// and the module can read and echo them back.
//
// The module:
// 1. Receives (ptr, len) pointing to JSON input in memory
// 2. Copies the input bytes to a new location (offset 2048)
// 3. Returns packed i64 pointing to the copy
func TestWasmProtocol_EchoInput(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	wasmBytes := buildEchoInputWasm(t)
	verifyWasmCompiles(t, wasmBytes)

	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "echo.wasm")
	os.WriteFile(wasmFile, wasmBytes, 0644)

	params := map[string]any{"args": []any{"hello", 123}}
	result, err := rt.Execute(context.Background(), wasmFile, "echo", params, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T: %v", result, result)
	}

	// The echo module returns the same JSON that was sent as input
	args, ok := resultMap["args"]
	if !ok {
		t.Fatalf("expected 'args' key in echoed result, got: %v", resultMap)
	}

	argsSlice, ok := args.([]any)
	if !ok {
		t.Fatalf("expected args to be []any, got %T", args)
	}
	if len(argsSlice) != 2 {
		t.Errorf("expected 2 args, got %d", len(argsSlice))
	}
}

// TestWasmProtocol_EmptyParams tests execution with nil/empty params.
func TestWasmProtocol_EmptyParams(t *testing.T) {
	rt := New(DefaultConfig())
	defer rt.Close()

	wasmBytes := buildEchoInputWasm(t)
	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "echo.wasm")
	os.WriteFile(wasmFile, wasmBytes, 0644)

	result, err := rt.Execute(context.Background(), wasmFile, "echo", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result for empty params echo")
	}
}

// TestWasmProtocol_PooledExecution tests the full protocol with instance pooling.
func TestWasmProtocol_PooledExecution(t *testing.T) {
	rt := New(Config{
		MaxMemoryMB:  64,
		MaxExecTime:  DefaultConfig().MaxExecTime,
		CompileCache: true,
		PoolSize:     2,
	})
	defer rt.Close()

	wasmBytes := buildFixedResponseWasm(t)
	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "pooled.wasm")
	os.WriteFile(wasmFile, wasmBytes, 0644)

	for i := 0; i < 5; i++ {
		result, err := rt.Execute(context.Background(), wasmFile, "process", map[string]any{"i": i}, nil)
		if err != nil {
			t.Fatalf("iteration %d: execute error: %v", i, err)
		}

		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("iteration %d: expected map, got %T", i, result)
		}
		if v, ok := resultMap["value"].(float64); !ok || v != 42 {
			t.Errorf("iteration %d: expected value=42, got %v", i, resultMap["value"])
		}
	}
}

// --- WASM module builders for protocol tests ---

// buildFixedResponseWasm creates a WASM module where the "process" function:
// - Ignores input
// - Writes '{"value":42}' (12 bytes) to memory at offset 2048
// - Returns packed i64: (2048 << 32) | 12
func buildFixedResponseWasm(t *testing.T) []byte {
	t.Helper()

	// The response JSON: {"value":42}
	responseJSON := []byte(`{"value":42}`)
	responseLen := len(responseJSON)

	// Build WASM with data section pre-populating memory at offset 2048
	return encodeModuleWithData(
		[]funcType{
			{params: []byte{i32}, results: []byte{i32}},      // malloc
			{params: []byte{i32, i32}, results: []byte{}},    // free
			{params: []byte{i32, i32}, results: []byte{i64}}, // process
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
			// malloc: return 1024
			{0x00, 0x41, 0x80, 0x08, 0x0b},
			// free: no-op
			{0x00, 0x0b},
			// process: return packed (2048 << 32) | responseLen
			buildReturnPackedI64(2048, uint32(responseLen)),
		},
		2048,
		responseJSON,
	)
}

// buildEchoInputWasm creates a WASM module where the "echo" function:
// - Receives (ptr, len) pointing to JSON input already in linear memory
// - Returns the same (ptr, len) as packed i64 — echoing the input back
// This works because the host already wrote the JSON to memory via malloc.
func buildEchoInputWasm(t *testing.T) []byte {
	t.Helper()

	return encodeModuleWithData(
		[]funcType{
			{params: []byte{i32}, results: []byte{i32}},      // malloc
			{params: []byte{i32, i32}, results: []byte{}},    // free
			{params: []byte{i32, i32}, results: []byte{i64}}, // echo
		},
		[]uint32{0, 1, 2},
		1,
		[]export{
			{name: "memory", kind: 0x02, index: 0},
			{name: "malloc", kind: 0x00, index: 0},
			{name: "free", kind: 0x00, index: 1},
			{name: "echo", kind: 0x00, index: 2},
		},
		[][]byte{
			// malloc: return 1024
			{0x00, 0x41, 0x80, 0x08, 0x0b},
			// free: no-op
			{0x00, 0x0b},
			// echo: return packed (param0 << 32) | param1
			buildEchoReturnInput(),
		},
		0, nil,
	)
}

// buildReturnPackedI64 generates WASM bytecode that returns a packed i64 constant.
func buildReturnPackedI64(ptr, length uint32) []byte {
	packed := (uint64(ptr) << 32) | uint64(length)
	var body []byte
	body = append(body, 0x00) // local count = 0
	body = append(body, 0x42) // i64.const
	body = append(body, encodeLEB128Signed(int64(packed))...)
	body = append(body, 0x0b) // end
	return body
}

// buildEchoReturnInput generates WASM bytecode that packs (param0 << 32) | param1 as i64.
func buildEchoReturnInput() []byte {
	var body []byte
	body = append(body, 0x00) // 0 locals

	// (i64.extend_i32_u param0) << 32
	body = append(body, 0x20, 0x00) // local.get $ptr (param 0)
	body = append(body, 0xad)       // i64.extend_i32_u
	body = append(body, 0x42, 0x20) // i64.const 32
	body = append(body, 0x86) // i64.shl

	// | (i64.extend_i32_u param1)
	body = append(body, 0x20, 0x01) // local.get $len (param 1)
	body = append(body, 0xad)       // i64.extend_i32_u
	body = append(body, 0x84)       // i64.or

	body = append(body, 0x0b) // end
	return body
}

// buildEchoFunctionBody generates WASM bytecode for the echo function.
// It copies bytes from (param0=src_ptr, param1=src_len) to offset 2048,
// then returns packed (2048 << 32) | src_len.
//
// Pseudocode:
//   local i = 0
//   while i < src_len:
//     memory[2048 + i] = memory[src_ptr + i]
//     i++
//   return (2048 << 32) | src_len
func buildEchoFunctionBody() []byte {
	var body []byte

	// 1 local: i (i32)
	body = append(body, 0x01)       // 1 local declaration
	body = append(body, 0x01, 0x7f) // 1 local of type i32

	// local.set $i = 0 (already 0 by default, skip)

	// block + loop for copy
	body = append(body, 0x02, 0x40) // block (void)
	body = append(body, 0x03, 0x40) // loop (void)

	// br_if 1 (break outer block) if i >= src_len
	body = append(body, 0x20, 0x02) // local.get $i
	body = append(body, 0x20, 0x01) // local.get $src_len (param 1)
	body = append(body, 0x4d)       // i32.ge_u
	body = append(body, 0x0d, 0x01) // br_if 1 (break to outer block)

	// memory[2048 + i] = memory[src_ptr + i]
	// i32.store8: store address = 2048 + i
	body = append(body, 0x41, 0x80, 0x10) // i32.const 2048
	body = append(body, 0x20, 0x02)       // local.get $i
	body = append(body, 0x6a)             // i32.add → dest addr

	// load: memory[src_ptr + i]
	body = append(body, 0x20, 0x00) // local.get $src_ptr (param 0)
	body = append(body, 0x20, 0x02) // local.get $i
	body = append(body, 0x6a)       // i32.add → src addr
	body = append(body, 0x2d, 0x00, 0x00) // i32.load8_u offset=0 align=0

	body = append(body, 0x3a, 0x00, 0x00) // i32.store8 offset=0 align=0

	// i++
	body = append(body, 0x20, 0x02) // local.get $i
	body = append(body, 0x41, 0x01) // i32.const 1
	body = append(body, 0x6a)       // i32.add
	body = append(body, 0x21, 0x02) // local.set $i

	// br 0 (continue loop)
	body = append(body, 0x0c, 0x00) // br 0

	body = append(body, 0x0b) // end loop
	body = append(body, 0x0b) // end block

	// return packed (2048 << 32) | src_len
	// i64.const 2048 << 32 = 8796093022208
	body = append(body, 0x42) // i64.const
	body = append(body, encodeLEB128Signed(int64(2048)<<32)...)

	// i64.or with (i64.extend_i32_u src_len)
	body = append(body, 0x20, 0x01) // local.get $src_len
	body = append(body, 0xad)       // i64.extend_i32_u
	body = append(body, 0x84)       // i64.or

	body = append(body, 0x0b) // end function

	return body
}

// encodeLEB128Signed encodes a signed 64-bit integer in LEB128 format.
func encodeLEB128Signed(v int64) []byte {
	var buf []byte
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if (v == 0 && b&0x40 == 0) || (v == -1 && b&0x40 != 0) {
			buf = append(buf, b)
			break
		}
		buf = append(buf, b|0x80)
	}
	return buf
}

// encodeModuleWithData extends encodeModule with an optional data section.
func encodeModuleWithData(types []funcType, funcIndices []uint32, memPages uint32, exports []export, codes [][]byte, dataOffset uint32, data []byte) []byte {
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

	// Data section (id=11) — optional
	if len(data) > 0 {
		dataSection := encodeDataSection(dataOffset, data)
		buf = append(buf, 0x0b)
		buf = append(buf, encodeU32(uint32(len(dataSection)))...)
		buf = append(buf, dataSection...)
	}

	return buf
}

func encodeDataSection(offset uint32, data []byte) []byte {
	var buf []byte
	buf = append(buf, encodeU32(1)...) // 1 data segment
	buf = append(buf, 0x00)            // memory index 0 (active)
	// offset expression: i32.const <offset>, end
	buf = append(buf, 0x41)
	buf = append(buf, encodeU32(offset)...)
	buf = append(buf, 0x0b) // end init expr
	buf = append(buf, encodeU32(uint32(len(data)))...)
	buf = append(buf, data...)
	return buf
}
