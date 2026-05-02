package wasm

import "github.com/tetratelabs/wazero/api"

func readStringFromMemory(m api.Module, ptr, length uint32) string {
	bytes, ok := m.Memory().Read(ptr, length)
	if !ok {
		return ""
	}
	return string(bytes)
}

func readBytesFromMemory(m api.Module, ptr, length uint32) []byte {
	bytes, ok := m.Memory().Read(ptr, length)
	if !ok {
		return nil
	}
	result := make([]byte, length)
	copy(result, bytes)
	return result
}

func packPtrLen(ptr, length uint32) uint64 {
	return (uint64(ptr) << 32) | uint64(length)
}

func unpackPtrLen(packed uint64) (ptr, length uint32) {
	return uint32(packed >> 32), uint32(packed & 0xFFFFFFFF)
}
