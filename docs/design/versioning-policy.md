# Versioning Policy

Each Image Policy will define a Versioning Policy. The Versioning Policy is what
helps the system decide which releases we want to be tracking.

For now, we'll follow the [SemVer](semver.org) format. This could potentially
change in the future.

All release types are treated equally.

## Release

The release type relates to an actual production release. This should be in the
form of `v1.2.3`.

## Release Candidate

Release Candidates are used for staging environments. These should be in the
form off `v1.2.3-rc.0`. This indicates that we can also have multiple release
candiates for the same release.

## Preview

Previews are development versions. They are usually associated with Pull
Requests and are tagged by a unique version, usually a commit sha.
