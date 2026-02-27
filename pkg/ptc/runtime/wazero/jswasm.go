package wazero

// QuickJS WASM Support (Future Implementation)
//
// This file is reserved for future QuickJS WASM integration.
// Currently, JavaScript execution in Wazero uses Goja internally
// because quickjs-emscripten requires Emscripten's JavaScript runtime
// which is complex to implement in pure Go.
//
// For JavaScript execution, both runtimes provide equivalent functionality:
//   - Goja runtime (--runtime goja): Pure Go JavaScript interpreter
//   - Wazero runtime (--runtime wazero): Uses Goja for JS, Wazero for WASM binaries
//
// Future improvements:
//   - Integrate a simpler QuickJS WASM build that doesn't require Emscripten
//   - Implement necessary Emscripten imports for quickjs-emscripten
//   - Use alternative JS engines compiled to pure WASM

// QuickJSInfo returns information about QuickJS WASM support
func QuickJSInfo() string {
	return `
QuickJS WASM Support
====================

Current Status:
- JavaScript execution uses Goja interpreter internally
- Wazero is used primarily for WASM binary execution

Future Plans:
- Direct QuickJS WASM integration for enhanced sandboxing
- Alternative: simpler QuickJS WASM build without Emscripten dependencies

For now, use:
  --runtime goja    # Recommended for JavaScript execution
  --runtime wazero  # For WASM binary execution
`
}
