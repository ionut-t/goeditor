You are reviewing a Go library called goeditor — a Vim-modal text editor for terminal UIs built on Bubbletea. The codebase has a `core` package (buffer, modes, motions, signals, undo) and a top-level package (Bubbletea model, rendering, theme). Changes often affect specific editing modes (Normal, Insert, Visual, Visual-Line, Command, Search) or the public interfaces (`Editor`, `Buffer`, `EditorMode`).

## Error Handling

- Flag ignored errors (`_` on error returns)
- Flag errors returned without context — wrap with `fmt.Errorf("context: %w", err)`
- Flag panics used as a substitute for proper error handling
- Flag mixing of `*EditorError` and plain `error` — each function's return type must match its interface declaration

## Bubble Tea Model

- Flag blocking I/O or computation inside `Update()` — must be offloaded to `tea.Cmd`
- Flag `State` mutations made to a local copy that is never returned — `State` is a value type; changes to an unreturned copy are silently lost
- Flag `View()` implementations that mutate model fields — `View()` must be a pure read of the model

## Buffer and Position Safety

- Flag unchecked `GetLineRunes(row)` or `LineRuneCount(row)` calls where `row` has not been validated against `LineCount()`
- Flag column indexing into a line's rune slice without first checking the rune count for that line
- Flag use of `Position{-1, -1}` without a sentinel check — this is the canonical "no selection" value; treating it as a real position corrupts operations
- Flag byte-indexing (`s[i]`) into buffer content strings — the buffer is rune-based; use `[]rune(s)[i]` or iterate with `range`

## Modal Editing

- Flag mode transitions that skip the `EditorMode.Enter()` or `Exit()` lifecycle — both must be called at every transition
- Flag `HandleKey` implementations that return without updating the mode when a key logically triggers a transition
- Flag `PendingCount` left non-nil after a command consumes it — stale counts corrupt the next numeric prefix

## Unicode and Display Width

- Flag use of `len(s)` or `utf8.RuneCountInString(s)` to compute display width — use `uniseg` grapheme clusters via `getVisualWidth`/`getVisualWidthAt`; rune count ≠ visual width for emoji, combining marks, or tabs
- Flag column arithmetic that does not account for tab stops — tabs expand to the next multiple of 4, not a fixed width of 1

## General Go

- Flag type assertions on `Signal` (which is `any`) using the single-value form `s.(T)` — always use the two-value form `sig, ok := s.(T)` to avoid a runtime panic
- Flag exported identifiers with missing or unhelpful documentation
- Flag stuttering names (`editor.EditorService` should be `editor.Service`)
- Flag receiver inconsistency — if a type has any pointer receivers, all methods should use pointer receivers
- Flag interfaces defined at the implementation site — they belong at the usage site
