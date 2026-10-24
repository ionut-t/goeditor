package core

import (
	"regexp"
	"strings"
)

// urlSchemeRe finds the start of a URL inside a WORD, as Vim's gx pattern
// does: \%(\%(http\|ftp\|irc\)s\?\|file\)://
var urlSchemeRe = regexp.MustCompile(`(?:(?:http|ftp|irc)s?|file)://`)

// openTargetUnderCursor returns what Vim's gx opens in normal mode: a URL in
// the WORD under the cursor, or else the file name under the cursor
// (<cfile>). It returns "" when there is neither.
func openTargetUnderCursor(buffer Buffer) string {
	pos := buffer.GetCursor().Position
	line := buffer.GetLineRunes(pos.Row)
	if url := urlInWORD(wordUnderCursor(line, pos.Col)); url != "" {
		return url
	}
	return fileNameInLine(line, pos.Col)
}

// wordUnderCursor returns Vim's <cWORD>: the run of non-blanks under col,
// or the next one on the line when col is on a blank.
func wordUnderCursor(line []rune, col int) []rune {
	col = max(col, 0)
	for col < len(line) && isWhiteSpace(line[col]) {
		col++
	}
	if col >= len(line) {
		return nil
	}
	start, end := col, col
	for start > 0 && !isWhiteSpace(line[start-1]) {
		start--
	}
	for end < len(line) && !isWhiteSpace(line[end]) {
		end++
	}
	return line[start:end]
}

// urlInWORD returns the URL in word, from its scheme to the end of the WORD
// minus any trailing characters other than ASCII letters, digits and '/'.
// Like Vim, this drops a closing paren that belongs to the URL and a
// non-ASCII tail.
func urlInWORD(word []rune) string {
	s := string(word)
	loc := urlSchemeRe.FindStringIndex(s)
	if loc == nil {
		return ""
	}
	return strings.TrimRightFunc(s[loc[0]:], func(r rune) bool {
		return !isASCIIAlnum(r) && r != '/'
	})
}

// fileNameInLine ports Vim's file_name_in_line(), which gives <cfile>.
func fileNameInLine(line []rune, col int) string {
	// Search forward for what could be the start of a file name.
	start := max(col, 0)
	for start < len(line) && !isFileNameChar(line[start]) {
		start++
	}
	if start >= len(line) {
		return ""
	}

	// Search backward for its first char, going back over the ':' of "://"
	// although ':' is not a file name char.
	for start > 0 && (isFileNameChar(line[start-1]) || isURLSeparator(line, start-1)) {
		start--
	}

	// Search forward for its last char. "\ " is part of the name, and after
	// "type://" so are ':', '?', '&' and '=' (http://host:8080?q=a&b=c).
	end := start
	inType, isURL := true, false
	for end < len(line) {
		r := line[end]
		escapedSpace := r == '\\' && end+1 < len(line) && line[end+1] == ' '
		inName := isFileNameChar(r) || escapedSpace || isURLSeparator(line, end) ||
			(isURL && strings.ContainsRune(":?&=", r))
		if !inName {
			break
		}
		if isASCIILetter(r) {
			if inType && isURLSeparator(line, end+1) {
				isURL = true
			}
		} else {
			inType = false
		}
		if escapedSpace {
			end++
		}
		end++
	}

	// Drop trailing punctuation, but not from "..", which could be a
	// directory name. Vim's length check counts bytes.
	name := line[start:end]
	if n := len(name); len(string(name)) > 2 && strings.ContainsRune(".,:;!", name[n-1]) && name[n-2] != '.' {
		name = name[:n-1]
	}
	return string(name)
}

// isFileNameChar reports whether r is in Vim's default Unix 'isfname'
// (@,48-57,/,.,-,_,+,,,#,$,%,~,=). Like Vim, it counts every non-ASCII
// character as part of a file name.
func isFileNameChar(r rune) bool {
	return r >= 0x80 || isASCIIAlnum(r) || strings.ContainsRune("/.-_+,#$%~=", r)
}

// isURLSeparator reports whether "://" starts at line[i].
func isURLSeparator(line []rune, i int) bool {
	return i >= 0 && i+2 < len(line) && line[i] == ':' && line[i+1] == '/' && line[i+2] == '/'
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isASCIIAlnum(r rune) bool {
	return isASCIILetter(r) || (r >= '0' && r <= '9')
}

// visualSelectionText returns the visual selection as Vim's gx sees it: the
// selected text of each line, joined with spaces.
func visualSelectionText(editor Editor, buffer Buffer) string {
	state := editor.GetState()
	start, end := NormalizeSelection(state.VisualStart, buffer.GetCursor().Position)
	parts := make([]string, 0, end.Row-start.Row+1)
	for row := start.Row; row <= end.Row; row++ {
		line := buffer.GetLineRunes(row)
		from, to := 0, len(line)
		if state.Mode != VisualLineMode {
			if row == start.Row {
				from = min(start.Col, len(line))
			}
			if row == end.Row {
				to = min(end.Col+1, len(line))
			}
		}
		parts = append(parts, string(line[from:max(from, to)]))
	}
	return strings.Join(parts, " ")
}
