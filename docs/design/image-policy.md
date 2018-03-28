# Image Policy

ImagePolicy is responsible for tracking the versions of an Image and making sure
the cluster is aware of the latest version that matches it's VersioningPolicy.

To do this, we'll have a controller set up which knows how to poll the registry
associated with the given image and secrets. This controller uses the CRD resync
mechanism to periodically check for image updates on the registry.

Once an update is found, the controller stores the information in the `status`
object associated with this CRD. This `status` object will be used by other
controllers to create other components for the application.

By utilising the status object, we can introduce a mechanism which allows us to
perform rollbacks and force push a specific version. It also means that
subsequent controllers don't need to depend on the functionality of ImagePolicy
controller and can just use the `status` object as the desired state.
