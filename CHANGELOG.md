# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `batox-timer` app: a countdown in a corner of the screen, on top of the game, that closes the game
  or powers the console off at zero. It turns amber in the last minute and red in the last 10
  seconds, and does nothing when stopped early. `-dry-run` tries it without acting.
- `ui.Countdown` component, `ui.Align` to place components on the left or right (corners, with
  `overlay.Top` or `overlay.Bottom`), and the `ui.Warning` and `ui.Danger` colors.
- `ui.Card` can be aligned to a side with `Align`.

### Added

- `overlay` package: shows images above everything on Batocera, fullscreen games included, through
  sway's wlr-layer-shell overlay layer. Pure Go, no CGO. `Show` for one image, `Run` for content
  that is repainted on a timer.
- `ui` package: BatoX theme, embedded fonts, text wrapping, rounded shapes and the `Card` component.
- `batox-message` app: shows a message card on top of the game for a few seconds.
