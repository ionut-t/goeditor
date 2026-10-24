# Changelog

## [0.13.0](https://github.com/ionut-t/goeditor/compare/v0.12.0...v0.13.0) (2026-10-24)


### Features

* add gx command to open URLs and files ([#91](https://github.com/ionut-t/goeditor/issues/91)) ([370b687](https://github.com/ionut-t/goeditor/commit/370b68782f670c633cb49f575369e73447f07d1d))

## [0.12.0](https://github.com/ionut-t/goeditor/compare/v0.11.5...v0.12.0) (2026-10-23)


### Features

* **core:** implement WORD text objects (iW and aW) ([#86](https://github.com/ionut-t/goeditor/issues/86)) ([033c2bc](https://github.com/ionut-t/goeditor/commit/033c2bcc6fb48a6cbaf737e3185ca680f0500d69))
* **core:** treat unmapped CTRL-H as backspace ([#89](https://github.com/ionut-t/goeditor/issues/89)) ([888178a](https://github.com/ionut-t/goeditor/commit/888178a87c62c57a1fc109c005b12e4c8e5de278))
* **visual:** add 'o'/'O' to swap visual selection ends ([#90](https://github.com/ionut-t/goeditor/issues/90)) ([7b63d8f](https://github.com/ionut-t/goeditor/commit/7b63d8f5048be6675c17f22ba015264499a2e0af))


### Bug Fixes

* **textobject:** correctly implement word text objects with counts and cross-line support ([#88](https://github.com/ionut-t/goeditor/issues/88)) ([24a45d1](https://github.com/ionut-t/goeditor/commit/24a45d10b5ce4e827ee1c3e6ab88d8a02f2f129a))

## [0.11.5](https://github.com/ionut-t/goeditor/compare/v0.11.4...v0.11.5) (2026-10-23)


### Bug Fixes

* **textobject:** skip the indent before a closing bracket in inner block objects ([#85](https://github.com/ionut-t/goeditor/issues/85)) ([4abd2c0](https://github.com/ionut-t/goeditor/commit/4abd2c0f5adbc52fb62451c1b64ae69f6cd4a597))
* **undo:** return the cursor to where the change began on undo and redo ([#83](https://github.com/ionut-t/goeditor/issues/83)) ([f4c5b60](https://github.com/ionut-t/goeditor/commit/f4c5b60da71d6404de0ee2d6e8d54c157954d6a9))

## [0.11.4](https://github.com/ionut-t/goeditor/compare/v0.11.3...v0.11.4) (2026-10-22)


### Bug Fixes

* **charsearch:** make backward search motions (F/T) exclusive ([#79](https://github.com/ionut-t/goeditor/issues/79)) ([f1696de](https://github.com/ionut-t/goeditor/commit/f1696deb1767107fbf3df7c6e3af3081f3dca1b9))
* **core:** align horizontal motion and preferred column with vim ([#77](https://github.com/ionut-t/goeditor/issues/77)) ([6d7b0f4](https://github.com/ionut-t/goeditor/commit/6d7b0f4c3af233bca54281c26a680145e2bbe7fc))
* **core:** align visual mode 'l' motion with Vim behaviour ([#75](https://github.com/ionut-t/goeditor/issues/75)) ([29390c4](https://github.com/ionut-t/goeditor/commit/29390c49487fcbdb1e6666cb9a469c08668e8842))
* **core:** apply Vim's w, e and exclusive-motion rules to operators ([#78](https://github.com/ionut-t/goeditor/issues/78)) ([d348c27](https://github.com/ionut-t/goeditor/commit/d348c27185e61393d740f9dd5f7f98d084a4d93f))
* **motion:** align paragraph motion ({ and }) with Vim behaviour ([#81](https://github.com/ionut-t/goeditor/issues/81)) ([d20bcc8](https://github.com/ionut-t/goeditor/commit/d20bcc853a46b5edb5c23163a9f66168a7c42f30))
* **operator:** correctly handle motions over an empty range ([#80](https://github.com/ionut-t/goeditor/issues/80)) ([8c7628d](https://github.com/ionut-t/goeditor/commit/8c7628d954d450981f9b60121a30327cc2a017f8))
* **textobject:** correctly apply operators to empty inner objects ([#82](https://github.com/ionut-t/goeditor/issues/82)) ([21aba17](https://github.com/ionut-t/goeditor/commit/21aba172da417be48592368a5eca31f8d32e4639))

## [0.11.3](https://github.com/ionut-t/goeditor/compare/v0.11.2...v0.11.3) (2026-10-22)


### Bug Fixes

* **clipboard:** wrap clipboard write error and use VisualLineMode constant ([#67](https://github.com/ionut-t/goeditor/issues/67)) ([0d6750a](https://github.com/ionut-t/goeditor/commit/0d6750a499d0a9e00a684dd0241ebe79669daf32))
* **core:** ensure setting empty content creates a single empty line ([#68](https://github.com/ionut-t/goeditor/issues/68)) ([7ed5f18](https://github.com/ionut-t/goeditor/commit/7ed5f18f07dee16be2c5cd2c35c7c0a103505507))
* **core:** make $ stick to the end of the line, as Vim's curswant MAXCOL ([#73](https://github.com/ionut-t/goeditor/issues/73)) ([bee0e69](https://github.com/ionut-t/goeditor/commit/bee0e696b9a4539d354139bc178529e07f6b0c60))
* **core:** make normal-mode &lt;Home&gt; go to column 0 ([#74](https://github.com/ionut-t/goeditor/issues/74)) ([84b31ec](https://github.com/ionut-t/goeditor/commit/84b31ecb7b0209c04cec00b6ab3c567030dc0572))
* **cursor:** align 'first non-blank' motion with Vim on blank lines ([#69](https://github.com/ionut-t/goeditor/issues/69)) ([ecc539b](https://github.com/ionut-t/goeditor/commit/ecc539b57923344c04edbd1c9ca6c984b5f03807))
* **render/motion:** improve tab rendering and backward word motion ([#70](https://github.com/ionut-t/goeditor/issues/70)) ([7498a06](https://github.com/ionut-t/goeditor/commit/7498a069bb6d934c32c6f8b8388d3d9291deee54))
* **render:** draw the cursor after a full last row at the start of the next row ([#72](https://github.com/ionut-t/goeditor/issues/72)) ([f3173e9](https://github.com/ionut-t/goeditor/commit/f3173e9d2a27e91535df2db0227803eb81796157))
* **render:** keep blanks at wrap points so every cursor column has a cell ([#71](https://github.com/ionut-t/goeditor/issues/71)) ([019fe5d](https://github.com/ionut-t/goeditor/commit/019fe5dd8f6ff85b2ff87ef422aba90849c94f1e))
* **search:** treat titlecase characters as uppercase in smartcase ([#63](https://github.com/ionut-t/goeditor/issues/63)) ([c97d0d2](https://github.com/ionut-t/goeditor/commit/c97d0d2e1f035592b4e26a4e25abf9e3a2960be9))
* **search:** use unicode case-folding for case-insensitive matching ([#65](https://github.com/ionut-t/goeditor/issues/65)) ([005953d](https://github.com/ionut-t/goeditor/commit/005953d215d80f226a619b36ca771abf5816aeeb))
* **search:** wrap inside Find for both directions ([#66](https://github.com/ionut-t/goeditor/issues/66)) ([9057112](https://github.com/ionut-t/goeditor/commit/9057112ca1a0a965700f35271fb9c1972e607a47))

## [0.11.2](https://github.com/ionut-t/goeditor/compare/v0.11.1...v0.11.2) (2026-10-20)


### Bug Fixes

* **completion:** prevent panic when inserting completion with stale cursor ([#58](https://github.com/ionut-t/goeditor/issues/58)) ([a00e743](https://github.com/ionut-t/goeditor/commit/a00e74302747acd1018773d2d056a0bda1e4ffc2))
* **core:** align :w and :wq command behaviour with Vim ([#57](https://github.com/ionut-t/goeditor/issues/57)) ([6fba344](https://github.com/ionut-t/goeditor/commit/6fba34495aacbe341eb80b32ce5862b801adb95b))
* **core:** correct line number command behaviour ([#60](https://github.com/ionut-t/goeditor/issues/60)) ([4686610](https://github.com/ionut-t/goeditor/commit/4686610e94f43ab26317aafcee90781352b2d6c3))
* **paste:** correct cursor position and count handling for paste operations ([#56](https://github.com/ionut-t/goeditor/issues/56)) ([42aca24](https://github.com/ionut-t/goeditor/commit/42aca246b134434e78330a3124804a99162463e9))
* **search:** correct parsing of \c and \C case modifiers ([#59](https://github.com/ionut-t/goeditor/issues/59)) ([b34ea4a](https://github.com/ionut-t/goeditor/commit/b34ea4ad891ca4471f5b00a421fabc9f61d1462c))
* **search:** ensure correct rune-based column indexing for multi-byte text ([#62](https://github.com/ionut-t/goeditor/issues/62)) ([5434eb3](https://github.com/ionut-t/goeditor/commit/5434eb3dadd17a66c1c2abe3cb0a246c91ac5754))
* **search:** honour smart case option during search ([#61](https://github.com/ionut-t/goeditor/issues/61)) ([cb0126e](https://github.com/ionut-t/goeditor/commit/cb0126e35477c86c2bde2f936519f116435d13ba))
* **visual:** normalise cursor state after visual operators ([#54](https://github.com/ionut-t/goeditor/issues/54)) ([089144f](https://github.com/ionut-t/goeditor/commit/089144fd4c76c8195a370e1a80bb7c4291a5d2d0))

## [0.11.1](https://github.com/ionut-t/goeditor/compare/v0.11.0...v0.11.1) (2026-10-19)


### Bug Fixes

* **motion:** align vertical movement on shorter lines with Vim behaviour ([#50](https://github.com/ionut-t/goeditor/issues/50)) ([b138efe](https://github.com/ionut-t/goeditor/commit/b138efe64509ecb7418fa2801993a725519db09a))
* **paste:** correctly handle paste failures from clipboard errors ([#52](https://github.com/ionut-t/goeditor/issues/52)) ([e6a7284](https://github.com/ionut-t/goeditor/commit/e6a72849357c7014ce838526a541b53135b070d8))

## [0.11.0](https://github.com/ionut-t/goeditor/compare/v0.10.1...v0.11.0) (2026-10-19)


### ⚠ BREAKING CHANGES

* Removed the deprecated KeyCtrlD, KeyCtrlU, KeyCtrlN, KeyCtrlP, KeyCtrlY and KeyCtrlE key codes (use Ctrl('d') or KeyEvent{Rune: 'd', Modifiers: ModCtrl}), Cursor.ScrollUp and Cursor.ScrollDown (use Editor.ScrollHalfPage), and Highlighter.InvalidateLine (use InvalidateCache).
* **core:** Cursor motion methods no longer take an availableWidth parameter. This affects MoveLeft, MoveRight, MoveUp, MoveDown, MoveDownToFirstNonBlank, MoveLeftOrUp, MoveRightOrDown, MoveToLineEnd, MoveToAfterLineEnd, MoveToFirstNonBlank, MoveToBufferEnd, MoveWordForward, MoveWordBackward, MoveWordToEnd and MoveWordToEndBackward.

### Bug Fixes

* **core:** keep the full column for j and k ([ccd80ff](https://github.com/ionut-t/goeditor/commit/ccd80ffe503c879d5c5e60bef5cb8cc9792b8c3b))


### Code Refactoring

* remove deprecated APIs and constants ([80f54aa](https://github.com/ionut-t/goeditor/commit/80f54aa5c5f26d164219c42a630247ea41e0305d))

## [0.10.1](https://github.com/ionut-t/goeditor/compare/v0.10.0...v0.10.1) (2026-10-19)


### Bug Fixes

* **core:** match Vim's whichwrap for h, l and arrow keys ([#46](https://github.com/ionut-t/goeditor/issues/46)) ([bb95689](https://github.com/ionut-t/goeditor/commit/bb95689f6e62b28e29f87ef3e7502822bd0a5e02))
* **core:** pass available width during horizontal cursor movements ([#43](https://github.com/ionut-t/goeditor/issues/43)) ([102f5f4](https://github.com/ionut-t/goeditor/commit/102f5f4addc6a31b1b43e226b2196a53704c69d5))
* **visual:** exit visual mode and update cursor position after yank ([#45](https://github.com/ionut-t/goeditor/issues/45)) ([63ea7b7](https://github.com/ionut-t/goeditor/commit/63ea7b7d2c4e1514d5cb1d21701ab58cecfeca18))

## [0.10.0](https://github.com/ionut-t/goeditor/compare/v0.9.0...v0.10.0) (2026-10-19)


### Features

* **core:** add support for insert counts and visual mode repeat ([#42](https://github.com/ionut-t/goeditor/issues/42)) ([ce96805](https://github.com/ionut-t/goeditor/commit/ce9680563a4e5d7c6e1f8e212683264e4daf395e))
* **core:** implement repeat change with dot command ([#38](https://github.com/ionut-t/goeditor/issues/38)) ([65a4052](https://github.com/ionut-t/goeditor/commit/65a40527c5b6ef98af15b98b816965297a289598))


### Bug Fixes

* **core:** clear pending count when editing commands are refused ([#40](https://github.com/ionut-t/goeditor/issues/40)) ([4131a54](https://github.com/ionut-t/goeditor/commit/4131a54cfb2cbf0f05357250315188f067c74678))
* **core:** match Vim for unbound Ctrl and Alt keys ([#41](https://github.com/ionut-t/goeditor/issues/41)) ([0708597](https://github.com/ionut-t/goeditor/commit/0708597062cda722ad265de682c1f1f3fa2a8e2b))

## [0.9.0](https://github.com/ionut-t/goeditor/compare/v0.8.2...v0.9.0) (2026-10-18)


### ⚠ BREAKING CHANGES

* **core:** hosts must update for the new undo API.
    - `Undo`, `Redo` and `UndoLine` return `[]core.TextEdit` instead of the buffer content.
    - `UndoSignal`/`RedoSignal`/`UndoLineSignal.Value()` return `[]TextEdit`.
    - `UndoMsg`/`RedoMsg`/`UndoLineMsg` carry `Edits` instead of `ContentBefore` (call `GetCurrentContent()` if you need the text).
    - `SaveHistory()` is removed: the editor closes undo steps itself.
    - `*EditorError` now implements `error`, so `Error()` returns a `string`; use `Err()` for the wrapped error.
    - `ErrorMsg.Error` holds the `*core.EditorError` itself rather than the error it wraps. The message is the same, and `errors.Is`/`errors.As` reach the wrapped error.

### Features

* **core:** return text edits from undo, redo and undo-line ([bca0f96](https://github.com/ionut-t/goeditor/commit/bca0f96c0470f22edf46250d8c9d414ae2292f85))
* **undo:** implement delta-based undo and redo system ([#33](https://github.com/ionut-t/goeditor/issues/33)) ([71de9ef](https://github.com/ionut-t/goeditor/commit/71de9ef1abc08b10044d50a3cdfe74543cc3b564))


### Bug Fixes

* **undo:** keep cursor inside the text after a corrupt undo ([#36](https://github.com/ionut-t/goeditor/issues/36)) ([968fd86](https://github.com/ionut-t/goeditor/commit/968fd8686f4177035a0d3b585c4e4545a9b88248))

## [0.8.2](https://github.com/ionut-t/goeditor/compare/v0.8.1...v0.8.2) (2026-10-17)


### Bug Fixes

* **core:** compose pre-operator and post-operator counts ([#29](https://github.com/ionut-t/goeditor/issues/29)) ([4ba0346](https://github.com/ionut-t/goeditor/commit/4ba034699b303ab90a656db5851861c6df4cfb0d))
* **core:** empty line range on linewise change operations ([#32](https://github.com/ionut-t/goeditor/issues/32)) ([14890e7](https://github.com/ionut-t/goeditor/commit/14890e7a4944bb8bb944618de520e8f45c162c9f))

## [0.8.1](https://github.com/ionut-t/goeditor/compare/v0.8.0...v0.8.1) (2026-10-16)


### Bug Fixes

* **charsearch:** preserve last character search across mode switches ([#26](https://github.com/ionut-t/goeditor/issues/26)) ([845a05a](https://github.com/ionut-t/goeditor/commit/845a05a57c268695cba862af3fe0abaaba0cb634))
* **core:** handle Enter key motion with count correctly ([#25](https://github.com/ionut-t/goeditor/issues/25)) ([fce3804](https://github.com/ionut-t/goeditor/commit/fce38045d384055cc5c9b6ad2dfd502220393692))
* **core:** handle line count for G motion and operators ([#23](https://github.com/ionut-t/goeditor/issues/23)) ([3e336c5](https://github.com/ionut-t/goeditor/commit/3e336c55f09b301ca1372aa2c0fffe8b24c3af9d))
* **core:** step cursor left when exiting insert mode ([#27](https://github.com/ionut-t/goeditor/issues/27)) ([de90309](https://github.com/ionut-t/goeditor/commit/de90309e989f173cd3866d2c1f6d118ba114cc96))

## [0.8.0](https://github.com/ionut-t/goeditor/compare/v0.7.0...v0.8.0) (2026-10-15)


### Features

* **scrolling:** add Vim-like scrolling commands and behaviour ([#14](https://github.com/ionut-t/goeditor/issues/14)) ([5dcd8e4](https://github.com/ionut-t/goeditor/commit/5dcd8e4d902f5e165280082f6067e75dd57b80a1))


### Bug Fixes

* **highlighter:** invalidate full token cache on content change ([#20](https://github.com/ionut-t/goeditor/issues/20)) ([4a0a10b](https://github.com/ionut-t/goeditor/commit/4a0a10bfb51dba74b46965d844dfce5f61b4cb38))
* **highlighter:** preserve style cache during token cache invalidation ([#16](https://github.com/ionut-t/goeditor/issues/16)) ([2d6c68d](https://github.com/ionut-t/goeditor/commit/2d6c68de36c17ad49efe2430dc0a19c75ad19e40))
* **highlighter:** stop tokeniseRange writing past its range ([#21](https://github.com/ionut-t/goeditor/issues/21)) ([ef1c572](https://github.com/ionut-t/goeditor/commit/ef1c5722038bee6f703e30f86d925de072df1bb3))
* **layout:** preserve tabs and non-space whitespace during line wrapping ([#17](https://github.com/ionut-t/goeditor/issues/17)) ([eacbfe4](https://github.com/ionut-t/goeditor/commit/eacbfe450cc05aeb689cdb827767497bd07d280a))


### Performance Improvements

* **layout:** optimise grapheme cluster iteration ([#19](https://github.com/ionut-t/goeditor/issues/19)) ([cccde1e](https://github.com/ionut-t/goeditor/commit/cccde1eabe6a6bdb3c94193d8f889937a54d344f))
* **layout:** optimise rune counting in visual layout generation ([#18](https://github.com/ionut-t/goeditor/issues/18)) ([ee414a9](https://github.com/ionut-t/goeditor/commit/ee414a93ae347d7b6bfc58e73a661f25ccfac463))

## [0.7.0](https://github.com/ionut-t/goeditor/compare/v0.6.1...v0.7.0) (2026-10-10)


### Features

* **editor:** add command line height control and shared status row ([#12](https://github.com/ionut-t/goeditor/issues/12)) ([c684944](https://github.com/ionut-t/goeditor/commit/c68494413089751e0d3bd617fa137dc05893ac48))

## [0.6.1](https://github.com/ionut-t/goeditor/compare/v0.6.0...v0.6.1) (2026-10-10)


### Bug Fixes

* **render:** ensure immediate redraw after programmatic state changes ([#10](https://github.com/ionut-t/goeditor/issues/10)) ([ff34a8b](https://github.com/ionut-t/goeditor/commit/ff34a8b60361fe04785a9aae3b9865c25eeea3c7))

## [0.6.0](https://github.com/ionut-t/goeditor/compare/v0.5.0...v0.6.0) (2026-08-15)


### ⚠ BREAKING CHANGES

* **core:** The `U` key in normal mode has been changed from redo to its standard Vim behaviour: undoing all recent changes on the last-changed line. Redo is now mapped to `<C-r>`.

### Features

* **core:** vim-style key remapping ([#6](https://github.com/ionut-t/goeditor/issues/6)) ([d668dc8](https://github.com/ionut-t/goeditor/commit/d668dc861c8660a509ba245fdc55a6433a5c401d))

## [0.5.0](https://github.com/ionut-t/goeditor/compare/v0.4.16...v0.5.0) (2026-07-25)


### Features

* **core:** implement missing vim motions and operators ([#3](https://github.com/ionut-t/goeditor/issues/3)) ([f0e4cbe](https://github.com/ionut-t/goeditor/commit/f0e4cbec6e9c542eb7ca4ab4e6ed060b13133ed8))
