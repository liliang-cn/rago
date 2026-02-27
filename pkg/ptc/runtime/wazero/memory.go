package wazero

import (
	"encoding/binary"

	"github.com/tetratelabs/wazero/api"
)

// readString reads a string from WASM memory
func readString(mem api.Memory, ptr, length uint32) string {
	if mem == nil || length == 0 {
		return ""
	}

	buf, ok := mem.Read(ptr, length)
	if !ok {
		return ""
	}
	return string(buf)
}

// writeString writes a string to WASM memory and returns pointer and length
func writeString(mem api.Memory, s string) (uint32, uint32) {
	if mem == nil || len(s) == 0 {
		return 0, 0
	}

	// Allocate memory for the string
	buf := []byte(s)
	length := uint32(len(buf))

	// Find free memory location
	// For simplicity, we use a fixed allocation area
	// In production, use a proper allocator
	ptr := uint32(0x10000) // Start at 64KB offset

	// Check if we can write there
	// Resize memory if needed
	memSize := mem.Size()
	needed := ptr + length + 1024 // Extra buffer
	if needed > memSize {
		// Can't grow memory easily, so use end of current memory
		ptr = memSize - length - 1
		if ptr < 0x10000 {
			ptr = 0x10000
		}
	}

	// Write the string
	mem.Write(ptr, buf)

	return ptr, length
}

// readBytes reads bytes from WASM memory
func readBytes(mem api.Memory, ptr, length uint32) []byte {
	if mem == nil || length == 0 {
		return nil
	}

	buf, ok := mem.Read(ptr, length)
	if !ok {
		return nil
	}
	return buf
}

// writeBytes writes bytes to WASM memory
func writeBytes(mem api.Memory, data []byte) uint32 {
	if mem == nil || len(data) == 0 {
		return 0
	}

	ptr := uint32(0x10000)
	mem.Write(ptr, data)
	return ptr
}

// readUint32 reads a uint32 from WASM memory
func readUint32(mem api.Memory, ptr uint32) uint32 {
	if mem == nil {
		return 0
	}

	buf, ok := mem.Read(ptr, 4)
	if !ok {
		return 0
	}
	return binary.LittleEndian.Uint32(buf)
}

// writeUint32 writes a uint32 to WASM memory
func writeUint32(mem api.Memory, ptr uint32, val uint32) {
	if mem == nil {
		return
	}

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, val)
	mem.Write(ptr, buf)
}

// readUint64 reads a uint64 from WASM memory
func readUint64(mem api.Memory, ptr uint32) uint64 {
	if mem == nil {
		return 0
	}

	buf, ok := mem.Read(ptr, 8)
	if !ok {
		return 0
	}
	return binary.LittleEndian.Uint64(buf)
}

// writeUint64 writes a uint64 to WASM memory
func writeUint64(mem api.Memory, ptr uint32, val uint64) {
	if mem == nil {
		return
	}

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, val)
	mem.Write(ptr, buf)
}

// MemoryAllocator provides simple bump allocation for WASM memory
type MemoryAllocator struct {
	base    uint32
	current uint32
	size    uint32
}

// NewMemoryAllocator creates a new memory allocator
func NewMemoryAllocator(base, size uint32) *MemoryAllocator {
	return &MemoryAllocator{
		base:    base,
		current: base,
		size:    size,
	}
}

// Allocate allocates memory and returns pointer
func (a *MemoryAllocator) Allocate(size uint32) uint32 {
	if a.current+size > a.base+a.size {
		return 0 // Out of memory
	}

	ptr := a.current
	a.current += size

	// Align to 4 bytes
	if a.current%4 != 0 {
		a.current += 4 - (a.current % 4)
	}

	return ptr
}

// Reset resets the allocator
func (a *MemoryAllocator) Reset() {
	a.current = a.base
}

// Free is a no-op for bump allocator
func (a *MemoryAllocator) Free(_ uint32) {
	// Bump allocator doesn't support free
}
