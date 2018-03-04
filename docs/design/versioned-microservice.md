# Versioned Microservice

Versioned Microservices represent the combined state of a Microservice at a
given point in time. It's a snapshot of how a certain Microservice should be
configured at a given moment in time.

A Versioned Microservice can be seen as the lowest level component of
Heighliner.

## Design Overview

The Versioned Microservice resource exists out of 2 parts, a Custom Resource
Definition and a Controller. Each of these have their own specific design goals
which are described below.

### Custom Resource Definition

As the Versioned Microservice is the lowest level component, it is very
declarative. It knows everything that is needed to know on how to deploy an
application. It also comes with a set of sensitive defaults for optional
configuration.

The goal of for these Versioned Microservice CRDs is to be completely managed by
a [Microservice](./microsesrvice.md) instead of manually defining these CRDs.

#### Metadata

As with all Kubernetes resources, Versioned Microservice accepts metadata.
`name`, `namespace` and `labels` will be inherited by the underlying components
which are created by the controller.

#### Specification

The specification for the Versioned Microservice CRD consists out of different
parts, each highlighted below. In the future, these parts will be pulled in from
other resources, linked together in the Microservice resource.

##### Availability

Availability defines a lot of parameters on how we should run our service in a
High Availability setup.

This will be automatically filled in through the [Availability Policy](./availability-policy.md)

```yaml
availability:
  replicas: 2
  minAvailable: 1
  maxUnavailable: 1
  restartPolicy: Always
  affinity:
    multiHost:
      weight: 100
    multiZone:
      weight: 90
  deploymentStrategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
```

##### Network

The Network specification is what we'll translate into Services and Ingresses.
Here we can define what ports a Microservice should use, what the domain name is
on which - if at all - it should be accessible and if it should use SSL or not.

This will be automatically filled in through the [Network Policy](./network-policy.md).

```yaml
network:
  ports:
  - name: headless
    externalPort: 80
    internalPort: 8080
  dns:
  - hostname: api.hlnr.io
    ttl: 3600
    tls: true
    port: headless
```

##### Volumes

This is a list of [Volumes](https://godoc.org/k8s.io/kubernetes/pkg/apis/core#Volume)

```yaml
volumes:
- name: jwt-public
  emptyDir: {}
```

##### Containers

This is a list of [Containers](https://godoc.org/k8s.io/kubernetes/pkg/apis/core#Container)
and can be configured at will.

This will be automatically generated from a different set of Policies, like the
[Image Policy](./image-policy.md), [Health Policy](./health-policy.md) and [Config Policy](./config-policy.md).

```yaml
containers:
- name: api
  image: hlnr.io/api:latest
  imagePullPolicy: IfNotPresent
  env:
  - name: PORT
    value: 8080
  readinessProbe:
    httpGet:
      path: /_healthz
      port: 8080
      initialDelaySeconds: 3
      periodSeconds: 3
  livenessProbe:
    httpGet:
      path: /_healthz
      port: 8080
      initialDelaySeconds: 5
      periodSeconds: 3
  resources: {}
  volumeMounts:
  - name: jwt-public
    mountPath: /jwt
    readOnly: true
```

### Controller

The Versioned Microservice Controller is a [Custom Controller](https://kubernetes.io/docs/concepts/api-extension/custom-resources/#custom-controllers) which takes
action to achieve the desired state of the cluster.

#### Responsibilities

The controller is responsible for a limited set of actions. It will delegate a
lot of its responsibilities to other components.

- fill in correct defaults to the CRD
- creating new components which form a microservice ([Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/), [Service](https://kubernetes.io/docs/concepts/services-networking/service/), [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), ...)
- apply updates to the created components
- cascade deletion of a resource

#### Exemptions

- keep track of changes to the underlying specifications, like secrets
- track multiple microservice versions and delete older versions
