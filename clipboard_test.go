package goeditor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWithClipboardNil checks that WithClipboard(nil) keeps the system
// clipboard instead of leaving the editor with none.
func TestWithClipboardNil(t *testing.T) {
	cfg := &modelConfig{clipboard: &clipboardImpl{}}
	WithClipboard(nil)(cfg)
	assert.IsType(t, &clipboardImpl{}, cfg.clipboard)

	cb := &memClipboard{}
	WithClipboard(cb)(cfg)
	assert.Same(t, cb, cfg.clipboard)
}
