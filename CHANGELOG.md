# CHANGELOG

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## Unreleased

### BREAKING CHANGES

- `ImagePolicy.Spec.ImagePullSecrets` is now under
  `ImagePolicy.Spec.ContainerRegistry.ImagePullSecrets`.

Example Before:

```
apiVersion: hlnr.io/v1alpha1
kind: ImagePolicy
spec:
  imagePullSecrets:
    - name: docker-registry
```

After:

```
apiVersion: hlnr.io/v1alpha1
kind: ImagePolicy
spec:
  containerRegistry:
    name: docker
    imagePullSecrets:
      - name: docker-registry
```

### Added

- Added a health check for the GitHub Callback Server.
- Added logging to indicate GH Callback Server requests.
- Added `ImagePolicy.Spec.ContainerRegistry` to specify which container registry
  to pull the image from.

### Fixed

- Fixed the Makefile target for generating files.
- Fixed a bug where the OwnerReference on a Ingress for the Service pointed to the wrong APIGroup.

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
