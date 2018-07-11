# Installation

In our installation process we've made a few assumptions, these are all about
what is already installed in your cluster. If your configuration is different,
it's still possible to install Heighliner, but you might have to change some of
the installation files found in [docs/kube](./kube).

We've assumed you have the following operators installed in your cluster:

- [Ingress Nginx](https://github.com/kubernetes/ingress-nginx) as an Ingress
- [Cert Manager](https://github.com/jetstack/cert-manager) for TLS Certificate management
- [External DNS](https://github.com/kubernetes-incubator/external-dns) for DNS configuration

## Manual Installation

We've templated our installation files so we can install things dynamically. The
key attributes that should be filled in are:

- **Version**: the version of Heighliner to install
- **GitHubAPIToken**: the API token to use when communicating with GitHub
- **DNSProvider**: the DNS provider that is set up with ExternalDNS

### GitHub

The GitHubCallbackURL is used to link the cluster with GitHub. This is further
described in [the GitHub Connector documentation](./design/github-connector.md#Domain).

We'll also need an [API Token](./design/github-connector.md#APIToken) which will
allow us to actually communicate with GitHub. This API Token is only needed when
you install a new GitHubRepository into your cluster.

### Applying the files

Once the attributes are filled in, we can go ahead and apply the files:

```
$ kubectl apply -f docs/kube
```

This will set up all the controllers and install the necessary RBAC rules. The
controllers will then install the CRDs accordingly.

Now that we have Heighliner up and running, we can start installing
Microservices.
