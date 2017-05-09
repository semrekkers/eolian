package lua

import (
	"sync"
	"testing"

	"buddin.us/eolian/module"

	assert "gopkg.in/go-playground/assert.v1"
)

func newVM(t *testing.T) *VM {
	init, err := module.Lookup("Direct")
	assert.Equal(t, err, nil)

	direct, err := init(nil)
	assert.Equal(t, err, nil)

	vm, err := NewVM(direct, &sync.Mutex{})
	assert.Equal(t, err, nil)

	err = vm.REPL()
	assert.Equal(t, err, nil)

	return vm
}
