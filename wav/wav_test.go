package wav

import (
	"testing"

	assert "gopkg.in/go-playground/assert.v1"
)

func TestFileLoad(t *testing.T) {
	w, err := Open("space_ghost_action.wav")
	defer w.Close()
	assert.Equal(t, err, nil)

	samples, err := w.ReadAll()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(samples), 26312)
}
