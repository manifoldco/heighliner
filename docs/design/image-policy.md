# Image Policy

ImagePolicy is responsible for tracking the versions of an Image and making sure
the cluster is aware of the latest version that matches it's VersioningPolicy.

ImagePolicies have Filters, these filters define where the releases will come
from. The ImagePolicy is then responsible for validating that the desired images
are available in the linked registry.

ImagePolicies can optionally define a match configuration. Match is used to
control how GitHub releases map to container registry images. By default, the
release name is directly mapped to an image tag name. You can use `from` and
`to` values to do pattern matching on the GitHub release, and templating to
match container image tags. You can also define container image labels to match
on in the same way.

The ImagePolicy also matches the releases with the Versioning Policy, this means
that there could be multiple releases available, but the ImagePolicy will filter
these out further to only select the ones that match the VersioningPolicy.
