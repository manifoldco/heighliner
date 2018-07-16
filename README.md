# Heighliner

[![Build Status](https://travis-ci.com/manifoldco/heighliner.svg?token=SbTMbCYMT5HWVmmTnBoj&branch=master)](https://travis-ci.com/manifoldco/heighliner)
[![codecov](https://codecov.io/gh/manifoldco/heighliner/branch/master/graph/badge.svg)](https://codecov.io/gh/manifoldco/heighliner)
[![Go Report Card](https://goreportcard.com/badge/github.com/manifoldco/heighliner)](https://goreportcard.com/report/github.com/manifoldco/heighliner)
[![GoDoc](https://godoc.org/github.com/manifoldco/heighliner?status.svg)](https://godoc.org/github.com/manifoldco/heighliner)

> A Heighliner is truly big. Its hold will tuck all of our frigates and transports
> into a little corner-we'll be just one small part of the ship's manifest.

Heighliner aims to make your workflow with GitHub and Kubernetes easy.

## Goals

**Cloud Native.** Instead of templating, Heighliner runs your infrastructure as
software, keeping the state of your deployments always as they should be.

**Connected.** The cluster is aware of container registry and source code
repository state. It reacts to them (creating new deploys), and reflects into
them (updating GitHub PR deployment status). Preview deploys are automatically
created and destroyed. Deploys can auto-update based on Semantic Versioning
policies, or be manually controlled.

**Complete.** A Heighliner Microservice comes with DNS and TLS out of the box.

**Convention and Configuration.** Reasonable defaults allow you to get up and
running without much effort, but can be overridded for customization.

## Installation

Heighliner consists out of multiple components, we've explained these in detail
in the [design docs](docs/design/README.md).

For a full installation process, have a look at the [installation docs](docs/installation.md).

## Usage

### Configure a GitHub Repository

Ensure that you have an API token installed in your cluster. Follow our [how to](docs/design/github-connector.md#APIToken)
for further instructions.

The GitHub repository resource is used to syncronize releases and pull requests
with cluster state, and update pull requests with deployment status.

```yaml
apiVersion: hlnr.io/v1alpha1
kind: GitHubRepository
metadata:
  name: cool-repository
spec:
  repo: my-repository
  owner: my-account
  configSecret:
    name: my-github-secret
```

### Configure a Versioning Policy

The versioning policy resource defines how microservices are updated based on
available releases.

```yaml
apiVersion: hlnr.io/v1alpha1
kind: VersioningPolicy
metadata:
  name: release-patch
spec:
  semVer:
    version: release
    level: patch
```

### Configure an Image Policy

The image policy resource syncronizes Docker container images with cluster
state. It cross references with GitHub releases, filtering out images that do
not match the versioning policy.

```yaml
apiVersion: hlnr.io/v1alpha1
kind: ImagePolicy
metadata:
  name: my-image-policy
spec:
  image: my-docker/my-image
  imagePullSecrets:
  - name: my-docker-secrets
  versioningPolicy:
    name: release-patch
  filter:
    github:
      name: cool-repository
```

### Configure a Network Policy

The network policy resource handles exposing instances of versioned
microservices within the cluster, or to the outside world. `domain` can be
templated for use with preview releases (pull requests).

```yaml
apiVersion: hlnr.io/v1alpha1
kind: NetworkPolicy
metadata:
  name: hlnr-www
spec:
  microservice:
    name: my-microservice
  ports:
  - name: headless
    port: 80
    targetPort: 80
  externalDNS:
  - domain: my-domain.com
    port: headless
    tlsGroup: my-cert-manager-tls-group
  updateStrategy:
    latest: {}
```

### Configure a  Microservice

The microservice resource is a template for deployments of images that match the
image policy.

```yaml
apiVersion: hlnr.io/v1alpha1
kind: Microservice
metadata:
  name: my-microservice
spec:
  imagePolicy:
    name: my-image-policy
```

## Contributing

Thanks for taking the time to join the community and helping out!

- Please familiarize yourself with the [Code of Conduct](./CODE_OF_CONDUCT.md)
  before contributing.
- Look at our [Contributing Guidelines](./CONTRIBUTING.md) for more infirmation
  about setting up your environment and how to contribute.
