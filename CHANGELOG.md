# CHANGELOG

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## Unreleased

### Added

- Added a health check for the GitHub Callback Server.
- Added logging to indicate GH Callback Server requests.

## [0.1.2] - 2018-07-16

### Fixed
- Fix image mapping by name only.
- Relax ImagePolicy CRD validation so it can be installed.
- Eliminate registry ping log message

## [0.1.1] - 2018-07-16

### Added
- Introduce an optional `match` field for ImagePolicy to control how releases
  map to container images, based on container image tag name, labels, or both.

## [0.1.0] - 2018-07-13

Docker Image: [arigato/heighliner:0.1.0](https://hub.docker.com/r/arigato/heighliner/tags)

Initial release
