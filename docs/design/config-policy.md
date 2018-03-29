# Config Policy

The Config Policy within Heighliner exists out of multiple parts and is supposed
to be pretty flexible.

## Definitions

We'll provide several ways to specify Config Policies for a resource. In this
section, we'll go over both possible solutions.

### CRDs

The first solution is using regular CRDs. Here we'll define a set of
configuration references like one could do on a Kubernetes Deployment. This
exists out of `ConfigMap`, `Secret` and `Volume`. When these sections are
defined within a Custom Resource Definition, it can be detected by the
Microservice controller and will be parsed into the right format for a
VersionedMicroservice.

### Annotations

It's also possible to set up `Secret` and `ConfigMap` resources individually and
annotate them with the `hlnr.io/config-policy: <name>` annotation. This will
then be pulled into an auto-generated ConfigPolicy which can then be used
by the Microservice controller.

This can be useful for when your secrets are automatically generated through
controllers like the [Manifold Credentials Controller](https://github.com/manifoldco/kubernetes-credentials).

*Note*: volumes have to be defined through CRDs.

### Mixture

It's also possible to mix both scenarios. This can be useful for combining
volumes and secrets.

## Controller

The Config Policy controller is responsible for watching all the associated
secrets with a definition and mark itself as being updated or not for the
Microservice controller. This will be done by setting up a `last-updated` status
which the Microservice controller can use to validate it's own state against.

This controller is also responsible for detecting new definitions based on
annotations.
