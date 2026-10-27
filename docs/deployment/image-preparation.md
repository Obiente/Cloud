# Preparing Swarm images

`scripts/deploy-swarm.sh --pull` and `scripts/deploy-swarm-dev.sh --pull` require
every requested image pull to succeed before changing host directories, node
labels, Docker configs or services. A failed pull stops the deployment. Correct
registry credentials or publish the missing image, then rerun the command.
Already downloaded images remain cached.

The production build and pull lists include `databases-service`; the database
control plane and database proxy use this same image.

`REGISTRY` defaults to `ghcr.io/obiente`. Set it to a registry and namespace, such
as `registry.example.com/team`, without a trailing slash. The standard, development
and HA manifests use the same registry as the scripts. Deployment scripts load
`.env` before constructing image names. `DASHBOARD_IMAGE` can override the complete
dashboard image reference.

`scripts/build-swarm.sh` accepts these settings from its environment and tags
images with the names consumed by the manifests. `BUILD_LOCAL=true` also builds
those names during a production deployment. Building on a manager does not make
images available to workers: push the images to the configured registry, or load
them onto every eligible node before deployment.

These checks validate image availability. Mutable `latest` tags do not establish
that every image belongs to one source revision. For a release, verify the image
revisions and use reviewed digests in the deployment manifest.
