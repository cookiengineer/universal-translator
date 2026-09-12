package bergamot

import "testing"

// Smoke test: verify the CGo bridge links and the native library loads by
// creating and destroying an engine without touching any model files.
func TestEngineLifecycle(t *testing.T) {
	engine, err := NewEngine(1, 0)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	defer engine.Close()
}
