package core

import (
	"bytes"
	"slices"
)

// Text arriving from outside while the user reads (a stream, a job's output)
// is added with AppendBytes, as Vim's out_io "buffer" adds a job's output:
// whole lines only, below the last line. Nothing the user owns moves, since
// an append never changes an existing character or row: the cursor, TopLine,
// mode, status line, visual anchor, undo and redo history, '.', and the
// modified state stay as they were. Old undo steps stay valid for the same
// reason. Unlike Vim, the append is not an undo step and the text counts as
// saved, so u never removes arrived text and :q isn't refused for it.

// savedAppender is implemented by buffers that can add lines to both their
// text and their saved content, so an append leaves IsModified exactly as it
// was. Buffers from hosts (SetBuffer) fall back to InsertRunesAt and
// SaveContent.
type savedAppender interface {
	appendSaved(text []rune, replaceEmpty bool)
}

// AppendBytes adds p at the end of the buffer as text that arrived from
// outside. Only whole lines are added: bytes after the last '\n' are held
// until a later call completes the line or FlushAppend is called. Each line
// goes below the current last line, except that the first append to a
// buffer holding one empty, unmodified line replaces that line. The cursor,
// TopLine, mode, status line, visual anchor, undo and redo history, '.', and
// IsModified are left as they were. SetContent and SetBuffer drop held bytes.
func (e *editor) AppendBytes(p []byte) {
	// Copies p, which the caller may reuse.
	data := append(e.appendTail, p...)
	i := bytes.LastIndexByte(data, '\n')
	if i < 0 {
		e.appendTail = data
		return
	}
	// '\n' never occurs inside a multi-byte UTF-8 sequence, so a rune split
	// across calls is always in the held part, with the rest of its line.
	e.appendLines(data[:i])
	if rest := data[i+1:]; len(rest) > 0 {
		e.appendTail = slices.Clone(rest) // don't keep the whole chunk alive
	} else {
		e.appendTail = nil
	}
}

// FlushAppend adds a held partial line, if any, as the last line (call it at
// EOF). The next AppendBytes starts a new line. An incomplete UTF-8 sequence
// at its end becomes U+FFFD, one per byte, as SetContent decodes it.
func (e *editor) FlushAppend() {
	if len(e.appendTail) == 0 {
		return
	}
	tail := e.appendTail
	e.appendTail = nil
	e.appendLines(tail)
}

// appendLines adds text, lines separated by '\n', below the last line,
// decoded as SetContent decodes a file. It edits the unwrapped buffer, so the
// edit isn't recorded for undo.
func (e *editor) appendLines(text []byte) {
	runes := bytes.Runes(text)
	buf := e.buffer.Buffer
	// Only the first append replaces an empty buffer's line, as in Vim: an
	// empty line that arrived leaves the buffer looking just as empty, and
	// replacing it again would make the result depend on how the stream was
	// cut into chunks.
	replace := !e.appended && buf.IsEmpty() && !buf.IsModified()
	e.appended = true
	if a, ok := buf.(savedAppender); ok {
		a.appendSaved(runes, replace)
		return
	}

	// A host buffer: insert, then mark the text saved if it was. The one
	// inexact case is undoing back to the saved text after an append, which
	// then reads as modified.
	cursor := buf.GetCursor()
	modified := buf.IsModified()
	if replace {
		_ = buf.InsertRunesAt(0, 0, runes) // can't fail at (0, 0)
	} else {
		last := buf.LineCount() - 1
		_ = buf.InsertRunesAt(last, buf.LineRuneCount(last), append([]rune{'\n'}, runes...))
	}
	if !modified {
		buf.SaveContent()
	}
	buf.SetCursor(cursor) // in case the buffer moves it; still valid
}

// appendSaved adds text, lines separated by '\n', below the last line and to
// the saved content, so IsModified doesn't change: a modified buffer stays
// modified, and undoing back to the saved text still reads as unmodified.
// With replaceEmpty the buffer holds one empty, unmodified line, which the
// text replaces, as in Vim.
func (b *textBuffer) appendSaved(text []rune, replaceEmpty bool) {
	var lines [][]rune
	start := 0
	for i, r := range text {
		if r == '\n' {
			lines = append(lines, text[start:i:i])
			start = i + 1
		}
	}
	lines = append(lines, text[start:len(text):len(text)])

	if replaceEmpty {
		b.lines = lines
		b.savedContent = string(text)
		return
	}
	b.lines = append(b.lines, lines...)
	b.savedContent += "\n" + string(text)
}
