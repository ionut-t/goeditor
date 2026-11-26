package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChangedTick(t *testing.T) {
	tests := []struct {
		name  string
		keys  string
		moved bool
	}{
		{"motions", "jjkwbG", false},
		{"visual selection", "vjl<Esc>", false},
		{"yank", "yyyw", false},
		{"search", "/world<CR>n", false},
		{"insert mode without typing", "i<Esc>", false},
		{"cancelled operator", "d<Esc>", false},
		{"delete line", "dd", true},
		{"typing", "ix<Esc>", true},
		{"replace char", "rx", true},
		{"undo", "xu", true},
		{"case operator", "gUU", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor("hello\nworld\nagain\nmore")
			before := e.ChangedTick()
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.moved, e.ChangedTick() != before)
		})
	}
}

func TestChangedTickMovesOnEveryChange(t *testing.T) {
	e := newTestEditor("hello\nworld")
	keys(e, 'x')
	steps := []struct {
		name   string
		change func()
	}{
		{"undo", func() { _, _ = e.Undo() }},
		{"redo", func() { _, _ = e.Redo() }},
		{"insert through GetBuffer", func() { _ = e.GetBuffer().InsertRunesAt(0, 0, []rune("x")) }},
		{"delete through GetBuffer", func() { _ = e.GetBuffer().DeleteRunesAt(0, 0, 1) }},
		{"SetContent through GetBuffer", func() { e.GetBuffer().SetContent([]byte("new")) }},
		{"SetContent", func() { e.SetContent([]byte("other")) }},
		{"SetBuffer", func() { e.SetBuffer(NewBufferFromBytes([]byte("third"))) }},
		{"AppendBytes", func() { e.AppendBytes([]byte("line\n")) }},
		{"FlushAppend", func() { e.AppendBytes([]byte("tail")); e.FlushAppend() }},
	}
	for _, s := range steps {
		before := e.ChangedTick()
		s.change()
		assert.NotEqualf(t, before, e.ChangedTick(), "%s should move ChangedTick", s.name)
	}
}

func TestChangedTickStaysWithoutChange(t *testing.T) {
	e := newTestEditor("hello")
	before := e.ChangedTick()
	e.AppendBytes([]byte("partial")) // no complete line yet
	_, _ = e.Undo()                  // nothing to undo
	_, _ = e.Redo()                  // nothing to redo
	_ = e.GetBuffer().InsertRunesAt(0, 0, nil)
	_ = e.GetBuffer().DeleteRunesAt(0, 5, 1) // past the end: removes nothing
	assert.Equal(t, before, e.ChangedTick())
}
