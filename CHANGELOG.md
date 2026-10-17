# Changelog

## [0.9.0](https://github.com/ionut-t/goeditor/compare/v0.8.2...v0.9.0) (2026-10-17)


### Features

* **undo:** implement delta-based undo and redo system ([#33](https://github.com/ionut-t/goeditor/issues/33)) ([71de9ef](https://github.com/ionut-t/goeditor/commit/71de9ef1abc08b10044d50a3cdfe74543cc3b564))

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
