# Heighliner

A Heighliner is truly big. Its hold will tuck all of our frigates and transports
into a little corner-we'll be just one small part of the ship's manifest.

## Goal

The goal of the Heighliner project is to streamline a deployment flow for
Kubernetes. Instead of having all the parts living outside the cluster, we've
moved them inside the cluster, making it truly Cloud Native.

## Installation

Heighliner exists out of multiple components, we've explained these in detail
in the [design docs](docs/design/README.md).

### Controllers

To install all the controllers, apply the YAML files in the `docs/kube`
directory.

**Note**: you'll want to update the callbacks URL in the `github-repository`
controller to a URL you'll use in your cluster.

### GitHub Token

First off, you'll need a GitHub token per repository you want to watch. This
should then be injected as a secret in your cluster, where the token key is
`GITHUB_AUTH_TOKEN`.

This secret should live in the same namespace as where you'd like the
applications to be deployed and where you install the CRDs. It will be used as
a Local Reference within your `GitHubRepository` CRD.
