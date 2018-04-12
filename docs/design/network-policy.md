# Network Policy

The NetworkPolicy defines if a Microservice needs to be discoverable through
either internal DNS or external DNS (or both). If defined, the NetworkPolicy
controller will create a Service for each deployed version of a Microservice
(VersionedMicroservice) that is specific for that version. This allows for
users to test their specific application by pointing it to a very specific
service.

When an UpdateStrategy is provided, we'll automatically create a global Service
for the application and point it to the correct version. This means that instead
of specifying a very specific internal domain, the application name can be used.

When a domain is set up, the NetworkPolicy will also link an Ingress to the
correct Service. This means that the external domain will always point to the
correct version depending on the provided UpdateStrategy.

## UpdateStrategies

The NetworkPolicy can have several UpdateStrategies. These Strategies are used
to set up internal and external DNS entries.

### Latest

Latest will use the `Released` attribute on a status Release of a Microservice
and pick the last version based on the name and link the Service to this
VersionedMicroservice.

### Manual

When Manual is selected, the NetworkPolicy controller won't do anything and use
the provided labels to point to the desired application.
