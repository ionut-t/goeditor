# Changelog

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
