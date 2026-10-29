# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `overlay` package: shows images above everything on Batocera, fullscreen games included, through
  sway's wlr-layer-shell overlay layer. Pure Go, no CGO. `Show` for one image, `Run` for content
  that is repainted on a timer.
- `ui` package: BatoX theme, embedded fonts, text wrapping, rounded shapes and the `Card` component.
- `batox-message` app: shows a message card on top of the game for a few seconds.
