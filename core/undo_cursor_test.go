package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Contents for the undo cursor tests. The final \n of undoLastEmpty ends the
// file, so its last line is empty.
const (
	undoHello     = "hello world"
	undoLines     = "one two\nthree four\nfive six"
	undoIndented  = "  abc def\n  ghi jkl\n  mno"
	undoOneLine   = "x"
	undoLastEmpty = "abc\n\n"
)

// undoCursorCase is a key sequence ending in u or <C-r>, and the text and
// cursor Vim 9.2 leaves after it. Vim's curswant is then the cursor's column.
type undoCursorCase struct {
	keys string // key notation
	text string
	want Position
}

func runUndoCursorCases(t *testing.T, initial string, tests []undoCursorCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(initial)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.text, content(e), "text")
			assert.Equal(t, tt.want, cursorPos(e), "cursor")
			assert.Equal(t, tt.want.Col, e.GetBuffer().GetCursor().Preferred, "Preferred")
		})
	}
}

// TestUndoCursorCharwise checks that u after a change within one line goes
// back to where the change began, as Vim's u_undoredo: the start of an
// operator's text (the earlier of the cursor and where the motion goes), the
// first character X deletes, or the cursor at the first edit (x, r, p,
// inserts), not the cursor before the command.
func TestUndoCursorCharwise(t *testing.T) {
	t.Run("hello", func(t *testing.T) {
		runUndoCursorCases(t, undoHello, []undoCursorCase{
			{"$dFwu", "hello world", Position{0, 6}},
			{"$dbu", "hello world", Position{0, 6}},
			{"$d0u", "hello world", Position{0, 0}},
			{"$xu", "hello world", Position{0, 10}},
			{"xu", "hello world", Position{0, 0}},
			{"3xu", "hello world", Position{0, 0}},
			{"w3xu", "hello world", Position{0, 6}},
			{"$Xu", "hello world", Position{0, 9}},
			{"$3Xu", "hello world", Position{0, 7}},
			{"wdwu", "hello world", Position{0, 6}},
			{"wdeu", "hello world", Position{0, 6}},
			{"dwu", "hello world", Position{0, 0}},
			{"wDu", "hello world", Position{0, 6}},
			{"wd$u", "hello world", Position{0, 6}},
			{"wC<Esc>u", "hello world", Position{0, 6}},
			{"wCxyz<Esc>u", "hello world", Position{0, 6}},
			{"wcwX<Esc>u", "hello world", Position{0, 6}},
			{"cwX<Esc>u", "hello world", Position{0, 0}},
			{"wrXu", "hello world", Position{0, 6}},
			{"w3rXu", "hello world", Position{0, 6}},
			{"w~u", "hello world", Position{0, 6}},
			{"w3~u", "hello world", Position{0, 6}},
			{"wiXY<Esc>u", "hello world", Position{0, 6}},
			{"$aXY<Esc>u", "hello world", Position{0, 10}},
			{"AXY<Esc>u", "hello world", Position{0, 10}},
			{"wiX<Esc>u", "hello world", Position{0, 6}},
			{"wywPu", "hello world", Position{0, 6}},
			{"wywpu", "hello world", Position{0, 6}},
			{"wyw0pu", "hello world", Position{0, 0}},
			{"wyw$pu", "hello world", Position{0, 10}},
			{"$yb0Pu", "hello world", Position{0, 0}},
			{"wdw0u", "hello world", Position{0, 6}},
			{"w2dwu", "hello world", Position{0, 6}},
			{"2dwu", "hello world", Position{0, 0}},
			{"d2wu", "hello world", Position{0, 0}},
			{"wdbu", "hello world", Position{0, 0}},
			{"$dTou", "hello world", Position{0, 8}},
			{"w$dFou", "hello world", Position{0, 7}},
			{"lldhu", "hello world", Position{0, 1}},
			{"wdiwu", "hello world", Position{0, 6}},
			{"wdawu", "hello world", Position{0, 5}},
			{"$diwu", "hello world", Position{0, 6}},
			{"wviwdu", "hello world", Position{0, 6}},
			{"wvdu", "hello world", Position{0, 6}},
			{"wgUiwu", "hello world", Position{0, 6}},
			{"wsX<Esc>u", "hello world", Position{0, 6}},
			{"w3sX<Esc>u", "hello world", Position{0, 6}},
			{"SX<Esc>u", "hello world", Position{0, 0}},
			{"wiX<Left>Y<Esc>u", "hello Xworld", Position{0, 6}},
			{"$gUbu", "hello world", Position{0, 6}},
			{"$g~0u", "hello world", Position{0, 0}},
			{"wgUwu", "hello world", Position{0, 6}},
			{"wgUUu", "hello world", Position{0, 0}},
			{"wg~~u", "hello world", Position{0, 0}},
			{"$vbUu", "hello world", Position{0, 6}},
			{"wv$~u", "hello world", Position{0, 6}},
			{"$vbuu", "hello world", Position{0, 6}},
			{"$vbrXu", "hello world", Position{0, 6}},
			{"$vb~u", "hello world", Position{0, 6}},
			{"w3ix<Esc>u", "hello world", Position{0, 6}},
			{"A<BS><BS>X<Esc>u", "hello world", Position{0, 10}},
			{"wi<BS><Esc>u", "hello world", Position{0, 6}},
			{"$vbc<Esc>u", "hello world", Position{0, 6}},
			{"$vbcX<Esc>u", "hello world", Position{0, 6}},
			{"$vbpu", "hello world", Position{0, 6}},
			{"wyw$vbpu", "hello world", Position{0, 6}},
		})
	})
}

// TestUndoCursorOverLines checks u after linewise and multi-line changes: dd,
// cc and S start at the earlier of the cursor and the first non-blank, o and
// O at the cursor before them, V at column 0 of its anchor or at the cursor.
// Not copied from Vim: after a linewise change over several lines (cj, 2cc,
// Vjc) Vim's u goes to the line below the first, as its op_change deletes
// those lines first; goeditor goes to the start.
func TestUndoCursorOverLines(t *testing.T) {
	t.Run("lines", func(t *testing.T) {
		runUndoCursorCases(t, undoLines, []undoCursorCase{
			{"ddu", "one two\nthree four\nfive six", Position{0, 0}},
			{"jddu", "one two\nthree four\nfive six", Position{1, 0}},
			{"jwddu", "one two\nthree four\nfive six", Position{1, 0}},
			{"Gddu", "one two\nthree four\nfive six", Position{2, 0}},
			{"Gwddu", "one two\nthree four\nfive six", Position{2, 0}},
			{"jw2ddu", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwdju", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwdku", "one two\nthree four\nfive six", Position{0, 6}},
			{"Ju", "one two\nthree four\nfive six", Position{0, 0}},
			{"wJu", "one two\nthree four\nfive six", Position{0, 4}},
			{"jwJu", "one two\nthree four\nfive six", Position{1, 6}},
			{"3Ju", "one two\nthree four\nfive six", Position{0, 0}},
			{"jw3Ju", "one two\nthree four\nfive six", Position{1, 6}},
			{"joX<Esc>u", "one two\nthree four\nfive six", Position{1, 0}},
			{"jwoX<Esc>u", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwOX<Esc>u", "one two\nthree four\nfive six", Position{1, 6}},
			{"GoX<Esc>u", "one two\nthree four\nfive six", Position{2, 0}},
			{"GwoX<Esc>u", "one two\nthree four\nfive six", Position{2, 5}},
			{"OX<Esc>u", "one two\nthree four\nfive six", Position{0, 0}},
			{"wOX<Esc>u", "one two\nthree four\nfive six", Position{0, 4}},
			{"jwoX<CR>Y<Esc>u", "one two\nthree four\nfive six", Position{1, 6}},
			{"yypu", "one two\nthree four\nfive six", Position{0, 0}},
			{"jwyypu", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwyyPu", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwyyGpu", "one two\nthree four\nfive six", Position{2, 0}},
			{"yy3pu", "one two\nthree four\nfive six", Position{0, 0}},
			{"wyyjjpu", "one two\nthree four\nfive six", Position{2, 4}},
			{"Gwyykkpu", "one two\nthree four\nfive six", Position{0, 5}},
			{"wyjpu", "one two\nthree four\nfive six", Position{0, 4}},
			{"jwyjPu", "one two\nthree four\nfive six", Position{1, 6}},
			{"wvjdu", "one two\nthree four\nfive six", Position{0, 4}},
			{"jwvjdu", "one two\nthree four\nfive six", Position{1, 6}},
			{"wvjy$pu", "one two\nthree four\nfive six", Position{0, 6}},
			{"wVjdu", "one two\nthree four\nfive six", Position{0, 0}},
			{"jwdggu", "one two\nthree four\nfive six", Position{0, 0}},
			{"wdGu", "one two\nthree four\nfive six", Position{0, 4}},
			{"jwd}u", "one two\nthree four\nfive six", Position{1, 6}},
			{"wd}u", "one two\nthree four\nfive six", Position{0, 4}},
			{"wiX<CR>Y<Esc>u", "one two\nthree four\nfive six", Position{0, 4}},
			{"A<CR>X<Esc>u", "one two\nthree four\nfive six", Position{0, 6}},
			{"jwi<CR><Esc>u", "one two\nthree four\nfive six", Position{1, 6}},
			{"jw$a<CR>X<Esc>u", "one two\nthree four\nfive six", Position{1, 9}},
			{"jwccX<Esc>u", "one two\nthree four\nfive six", Position{1, 0}},
			{"jwCX<Esc>u", "one two\nthree four\nfive six", Position{1, 6}},
			{"jw$d0u", "one two\nthree four\nfive six", Position{1, 0}},
			{"jwdbu", "one two\nthree four\nfive six", Position{1, 0}},
			{"j0dbu", "one two\nthree four\nfive six", Position{0, 4}},
			{"jwdgeu", "one two\nthree four\nfive six", Position{1, 4}},
			{"jxu", "one two\nthree four\nfive six", Position{1, 0}},
			{"jwxu", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwd2wu", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwd3wu", "one two\nthree four\nfive six", Position{1, 6}},
			{"wd3wu", "one two\nthree four\nfive six", Position{0, 4}},
			{"jw$dwu", "one two\nthree four\nfive six", Position{1, 9}},
			{"2ddu", "one two\nthree four\nfive six", Position{0, 0}},
			{"jwvjcX<Esc>u", "one two\nthree four\nfive six", Position{1, 6}},
			{"jw3oX<Esc>u", "one two\nthree four\nfive six", Position{1, 6}},
			{"jw3OX<Esc>u", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwgUju", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwVjUu", "one two\nthree four\nfive six", Position{1, 0}},
			{"GwVkUu", "one two\nthree four\nfive six", Position{1, 5}},
			{"jwvj~u", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwvjJu", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwVjJu", "one two\nthree four\nfive six", Position{1, 0}},
			{"Gwyyggpu", "one two\nthree four\nfive six", Position{0, 0}},
			{"wyiwjj$pu", "one two\nthree four\nfive six", Position{2, 7}},
			{"jwvjy0Pu", "one two\nthree four\nfive six", Position{1, 0}},
			{"jwVjyPu", "one two\nthree four\nfive six", Position{1, 0}},
			{"jwVjyGpu", "one two\nthree four\nfive six", Position{2, 0}},
			{"jwDu", "one two\nthree four\nfive six", Position{1, 6}},
			{"jw2Du", "one two\nthree four\nfive six", Position{1, 6}},
			{"jw2C<Esc>u", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwvjpu", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwi<CR><CR><Esc>u", "one two\nthree four\nfive six", Position{1, 6}},
		})
	})
	t.Run("indented", func(t *testing.T) {
		runUndoCursorCases(t, undoIndented, []undoCursorCase{
			{"IXY<Esc>u", "  abc def\n  ghi jkl\n  mno", Position{0, 2}},
			{"wIXY<Esc>u", "  abc def\n  ghi jkl\n  mno", Position{0, 2}},
			{"wd0u", "  abc def\n  ghi jkl\n  mno", Position{0, 0}},
			{"w^d$u", "  abc def\n  ghi jkl\n  mno", Position{0, 2}},
			{"$d^u", "  abc def\n  ghi jkl\n  mno", Position{0, 2}},
			{"jwddu", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"jwJu", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"jwoX<Esc>u", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"jwOX<Esc>u", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"jwyypu", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"jwyyPu", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"jw2ddu", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"Gwddu", "  abc def\n  ghi jkl\n  mno", Position{2, 2}},
			{"Gwdku", "  abc def\n  ghi jkl\n  mno", Position{1, 4}},
			{"wdju", "  abc def\n  ghi jkl\n  mno", Position{0, 2}},
			{"jwdju", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"jwdku", "  abc def\n  ghi jkl\n  mno", Position{0, 2}},
			{"jwSX<Esc>u", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"jwccX<Esc>u", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"jwVdu", "  abc def\n  ghi jkl\n  mno", Position{1, 0}},
			{"jw>>u", "  abc def\n  ghi jkl\n  mno", Position{1, 2}},
			{"Gwdggu", "  abc def\n  ghi jkl\n  mno", Position{0, 2}},
			{"wdGu", "  abc def\n  ghi jkl\n  mno", Position{0, 2}},
			{"ccX<Esc>u", "  abc def\n  ghi jkl\n  mno", Position{0, 0}},
		})
	})
	t.Run("oneLine", func(t *testing.T) {
		runUndoCursorCases(t, undoOneLine, []undoCursorCase{
			{"ddu", "x", Position{0, 0}},
			{"xu", "x", Position{0, 0}},
			{"oX<Esc>u", "x", Position{0, 0}},
		})
	})
	t.Run("lastEmpty", func(t *testing.T) {
		runUndoCursorCases(t, undoLastEmpty, []undoCursorCase{
			{"Gddu", "abc\n", Position{1, 0}},
			{"GiX<Esc>u", "abc\n", Position{1, 0}},
		})
	})
}

// TestUndoCursorAfterRepeats checks u after '.', and several u in a row: each
// step goes back to where its own change began.
func TestUndoCursorAfterRepeats(t *testing.T) {
	t.Run("hello", func(t *testing.T) {
		runUndoCursorCases(t, undoHello, []undoCursorCase{
			{"wiX<Left>Y<Esc>uu", "hello world", Position{0, 6}},
			{"x.u", "ello world", Position{0, 0}},
			{"x.uu", "hello world", Position{0, 0}},
			{"wdw.u", "hello ", Position{0, 5}},
			{"dww.u", "world", Position{0, 4}},
			{"wiX<Esc>w.u", "hello Xworld", Position{0, 11}},
			{"x$.u", "ello world", Position{0, 9}},
			{"3x.u", "lo world", Position{0, 0}},
			{"x3.u", "ello world", Position{0, 0}},
			{"$dFw.u", "hello world", Position{0, 6}},
			{"xwx$xuuu", "hello world", Position{0, 0}},
			{"$x0xwxuuu", "hello world", Position{0, 10}},
			{"$x0xuuu", "hello world", Position{0, 10}},
		})
	})
	t.Run("lines", func(t *testing.T) {
		runUndoCursorCases(t, undoLines, []undoCursorCase{
			{"dd.u", "three four\nfive six", Position{0, 0}},
			{"jwdd.u", "one two\nfive six", Position{1, 0}},
			{"jwoX<Esc>.u", "one two\nthree four\nX\nfive six", Position{2, 0}},
			{"J.u", "one two three four\nfive six", Position{0, 7}},
			{"jwxj.u", "one two\nthree our\nfive six", Position{2, 6}},
			{"wxj.k.u", "one wo\nthre four\nfive six", Position{0, 4}},
			{"jwyyp.u", "one two\nthree four\nthree four\nfive six", Position{2, 0}},
			{"xj$.u", "ne two\nthree four\nfive six", Position{1, 9}},
			{"jddGxuu", "one two\nthree four\nfive six", Position{1, 0}},
			{"Gwxggwdduu", "one two\nthree four\nfive six", Position{2, 5}},
			{"jwxGwoX<Esc>uu", "one two\nthree four\nfive six", Position{1, 6}},
			{"xjwdduu", "one two\nthree four\nfive six", Position{0, 0}},
			{"jwJGxuu", "one two\nthree four\nfive six", Position{1, 6}},
			{"jwyypGwxuu", "one two\nthree four\nfive six", Position{1, 6}},
			{"xwxjxuuu", "one two\nthree four\nfive six", Position{0, 0}},
			{"GddGdduu", "one two\nthree four\nfive six", Position{2, 0}},
			{"dddddduuu", "one two\nthree four\nfive six", Position{0, 0}},
		})
	})
}

// TestRedoCursor checks <C-r>. As Vim, redo goes back to where the change began
// too (not to where it ended), on the first non-blank when the change removed
// that line.
func TestRedoCursor(t *testing.T) {
	t.Run("hello", func(t *testing.T) {
		runUndoCursorCases(t, undoHello, []undoCursorCase{
			{"wdwu<C-r>", "hello ", Position{0, 5}},
			{"$dFwu<C-r>", "hello d", Position{0, 6}},
			{"xu<C-r>", "ello world", Position{0, 0}},
			{"$xu<C-r>", "hello worl", Position{0, 9}},
			{"wiXY<Esc>u<C-r>", "hello XYworld", Position{0, 6}},
			{"xwxuu<C-r>", "ello world", Position{0, 0}},
			{"xwxuu<C-r><C-r>", "ello orld", Position{0, 5}},
			{"wdwu0<C-r>", "hello ", Position{0, 5}},
			{"$d0u<C-r>", "d", Position{0, 0}},
			{"wywPu<C-r>", "hello worldworld", Position{0, 6}},
			{"wywpu<C-r>", "hello wworldorld", Position{0, 6}},
			{"wcwX<Esc>u<C-r>", "hello X", Position{0, 6}},
			{"w3ix<Esc>u<C-r>", "hello xxxworld", Position{0, 6}},
			{"$vbdu<C-r>", "hello ", Position{0, 5}},
			{"wDu<C-r>", "hello ", Position{0, 5}},
			{"$x$x$xuuu<C-r><C-r><C-r>", "hello wo", Position{0, 7}},
		})
	})
	t.Run("lines", func(t *testing.T) {
		runUndoCursorCases(t, undoLines, []undoCursorCase{
			{"jddu<C-r>", "one two\nfive six", Position{1, 0}},
			{"jwddu<C-r>", "one two\nfive six", Position{1, 0}},
			{"Gddu<C-r>", "one two\nthree four", Position{1, 0}},
			{"jwoX<Esc>u<C-r>", "one two\nthree four\nX\nfive six", Position{1, 6}},
			{"jwOX<Esc>u<C-r>", "one two\nX\nthree four\nfive six", Position{1, 0}},
			{"Ju<C-r>", "one two three four\nfive six", Position{0, 0}},
			{"jwJu<C-r>", "one two\nthree four five six", Position{1, 6}},
			{"jwyypu<C-r>", "one two\nthree four\nthree four\nfive six", Position{1, 6}},
			{"jwyyPu<C-r>", "one two\nthree four\nthree four\nfive six", Position{1, 6}},
			{"wvjdu<C-r>", "one  four\nfive six", Position{0, 4}},
			{"jw2ddu<C-r>", "one two", Position{0, 0}},
			{"Gwxggdduu<C-r><C-r>", "three four\nfive ix", Position{0, 0}},
			{"jwxuG<C-r>", "one two\nthree our\nfive six", Position{1, 6}},
			{"wiX<CR>Y<Esc>u<C-r>", "one X\nYtwo\nthree four\nfive six", Position{0, 4}},
			{"Gwdggu<C-r>", "", Position{0, 0}},
			{"jwdGu<C-r>", "one two", Position{0, 0}},
			{"jwdku<C-r>", "five six", Position{0, 6}},
			{"jwVjdu<C-r>", "one two", Position{0, 0}},
			{"jw3Ju<C-r>", "one two\nthree four five six", Position{1, 6}},
			{"Gwyyggpu<C-r>", "one two\nfive six\nthree four\nfive six", Position{0, 0}},
			{"jwvjyGpu<C-r>", "one two\nthree four\nffour\nfive siive six", Position{2, 0}},
			{"dddddduuu<C-r><C-r><C-r>", "", Position{0, 0}},
			{"jwi<CR><Esc>u<C-r>", "one two\nthree \nfour\nfive six", Position{1, 5}},
			{"jwoX<Esc>ggxuu<C-r><C-r>", "ne two\nthree four\nX\nfive six", Position{0, 0}},
		})
	})
	t.Run("indented", func(t *testing.T) {
		runUndoCursorCases(t, undoIndented, []undoCursorCase{
			{"jwddu<C-r>", "  abc def\n  mno", Position{1, 2}},
			{"jwJu<C-r>", "  abc def\n  ghi jkl mno", Position{1, 2}},
			{"jwoX<Esc>u<C-r>", "  abc def\n  ghi jkl\nX\n  mno", Position{1, 2}},
			{"jwyypu<C-r>", "  abc def\n  ghi jkl\n  ghi jkl\n  mno", Position{1, 2}},
			{"jw2ddu<C-r>", "  abc def", Position{0, 2}},
			{"Gwddu<C-r>", "  abc def\n  ghi jkl", Position{1, 2}},
			{"jwccX<Esc>u<C-r>", "  abc def\nX\n  mno", Position{1, 0}},
		})
	})
	t.Run("oneLine", func(t *testing.T) {
		runUndoCursorCases(t, undoOneLine, []undoCursorCase{
			{"ddu<C-r>", "", Position{0, 0}},
		})
	})
	t.Run("lastEmpty", func(t *testing.T) {
		runUndoCursorCases(t, undoLastEmpty, []undoCursorCase{
			{"Gddu<C-r>", "abc", Position{0, 0}},
		})
	})
}
