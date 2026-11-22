# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.3.0] - 2026-11-22

### Changed

- A new look for the message card and the countdown, the one of the BatoX web app and of
  [batox-theme](https://github.com/monteirohat/batox-theme): a dark indigo surface with square
  corners in a thin white frame, in place of the rounded card with a gradient bar; the title is
  a label in small capitals; the countdown's digits are large and light over a thin progress
  line, and near the end the digits, the line and the frame turn amber, then red.
- The embedded fonts are Inter Light, Regular and SemiBold (Latin, Greek and Cyrillic) in place
  of the Go fonts. `ui.Light` and `ui.SemiBold` are new; `ui.Bold` now means SemiBold.
- `ui.Theme` has `Frame` and `Track` in place of `Accent`. New helpers: `ui.FillRect`,
  `ui.FillFramedRect`, `ui.Label`, `ui.CapHeight`, `ui.MeasureSpaced`, `ui.DrawTextSpaced`,
  `ui.MeasureDigits` and `ui.DrawDigits` (a number whose digits change without shifting).

## [0.2.0] - 2026-10-30

### Added

- `batox-timer` app: a countdown in a corner of the screen, on top of the game, that closes the game
  or powers the console off at zero. It turns amber in the last minute and red in the last 10
  seconds, and does nothing when stopped early. `-dry-run` tries it without acting.
- `ui.Countdown` component, `ui.Align` to place components on the left or right (corners, with
  `overlay.Top` or `overlay.Bottom`), and the `ui.Warning` and `ui.Danger` colors.
- `ui.Card` can be aligned to a side with `Align`.

## [0.1.0] - 2026-10-29

### Added

- `overlay` package: shows images above everything on Batocera, fullscreen games included, through
  sway's wlr-layer-shell overlay layer. Pure Go, no CGO. `Show` for one image, `Run` for content
  that is repainted on a timer.
- `ui` package: BatoX theme, embedded fonts, text wrapping, rounded shapes and the `Card` component.
- `batox-message` app: shows a message card on top of the game for a few seconds.

[Unreleased]: https://github.com/monteirohat/batox-apps/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/monteirohat/batox-apps/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/monteirohat/batox-apps/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/monteirohat/batox-apps/releases/tag/v0.1.0
