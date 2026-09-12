// Package bergamot is the CGo adapter that bridges the Go application to the
// native Bergamot/Marian translation engine through a thin C ABI.
package bergamot

/*
#cgo CFLAGS: -I${SRCDIR}/../../native/bridge
#cgo LDFLAGS: -L${SRCDIR}/../../native/build -lbergamot_bridge -Wl,-rpath,${SRCDIR}/../../native/build -Wl,-rpath,$ORIGIN

#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

// Engine owns a native translation service and the shared state required to
// turn its asynchronous interface into a blocking one. An Engine is not safe
// for concurrent use; the caller must serialise access to it.
type Engine struct {
	handle *C.BergamotBridge
}

// Model is an opaque handle to a loaded translation model. A model is bound to
// a single language direction.
type Model struct {
	handle *C.BergamotTranslationModel
}

// NewEngine creates a translation service with the given number of worker
// threads and cache size (0 disables the cache).
func NewEngine(numWorkers int, cacheSize int) (*Engine, error) {
	handle := C.bergamot_bridge_create(C.size_t(numWorkers), C.size_t(cacheSize))
	if handle == nil {
		return nil, errors.New("could not create the bergamot translation service")
	}

	engine := &Engine{handle: handle}
	runtime.SetFinalizer(engine, (*Engine).Close)
	return engine, nil
}

// LoadModel loads a translation model from a Marian configuration file
// (e.g. config.intgemm8bitalpha.yml) inside a model directory.
func (engine *Engine) LoadModel(configPath string) (*Model, error) {
	cPath := C.CString(configPath)
	defer C.free(unsafe.Pointer(cPath))

	handle := C.bergamot_bridge_load_model(engine.handle, cPath)
	if handle == nil {
		return nil, errors.New(engine.LastError())
	}

	model := &Model{handle: handle}
	runtime.SetFinalizer(model, (*Model).Close)
	return model, nil
}

// Close releases the native model handle.
func (model *Model) Close() {
	if model.handle != nil {
		C.bergamot_bridge_unload_model(model.handle)
		model.handle = nil
	}
}

// Translate runs a direct translation using a single model.
func (engine *Engine) Translate(model *Model, sourceText string) (string, error) {
	return engine.run(model, nil, sourceText)
}

// Pivot runs a two-step translation (source -> pivot, pivot -> target) using
// two models whose intermediate language must match.
func (engine *Engine) Pivot(first *Model, second *Model, sourceText string) (string, error) {
	return engine.run(first, second, sourceText)
}

func (engine *Engine) run(first *Model, second *Model, sourceText string) (string, error) {
	cText := C.CString(sourceText)
	defer C.free(unsafe.Pointer(cText))

	var output *C.char
	var status C.int

	if second == nil {
		status = C.bergamot_bridge_translate(engine.handle, first.handle, cText, &output)
	} else {
		status = C.bergamot_bridge_pivot(engine.handle, first.handle, second.handle, cText, &output)
	}

	if status != 0 {
		return "", errors.New(engine.LastError())
	}
	defer C.bergamot_bridge_free_string(output)

	return C.GoString(output), nil
}

// Cancel interrupts an in-progress translation.
func (engine *Engine) Cancel() {
	C.bergamot_bridge_cancel(engine.handle)
}

// Close releases the native service handle.
func (engine *Engine) Close() {
	if engine.handle != nil {
		C.bergamot_bridge_destroy(engine.handle)
		engine.handle = nil
	}
}

// LastError returns the last error recorded by the native engine.
func (engine *Engine) LastError() string {
	return C.GoString(C.bergamot_bridge_last_error(engine.handle))
}
