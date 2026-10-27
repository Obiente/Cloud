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

## Forced restarts

`deploy-swarm.sh --force` waits for each selected application service, including
the database control plane and proxy, to converge. `force-deploy.sh` restarts every
service in the stack once, sequentially. Both commands exit unsuccessfully if
Docker cannot list the services or if any requested update fails or exceeds its
deadline.

`SWARM_UPDATE_TIMEOUT_SECONDS` defaults to 300 seconds per service. Increase it
for services that legitimately need longer to converge. A timeout stops waiting;
Docker may continue the update. Inspect the named services with `docker service ps`
and check their logs before retrying.

## Dashboard readiness

The main deployment and dashboard redeployment commands wait for all dashboard
replicas to run the requested image, for any rolling update to finish, and for
`DASHBOARD_URL` to return HTTP 200. A paused or rolled-back update fails immediately.
Missing replicas, Docker outages and unreachable HTTP routes fail after
`DASHBOARD_READY_TIMEOUT_SECONDS` (300 seconds by default). The route must be
reachable from the machine running the deployment command.

Dashboard redeployment allows Traefik to discover the service through its normal
Swarm provider updates; it does not restart the shared proxy. The dashboard itself
restarts on any exit without a finite retry limit so a prolonged dependency outage
does not permanently exhaust its recovery attempts.

## Rollback results

`rollback-swarm.sh` waits for each requested service rollback and returns a failed
exit status if any service fails. A missing previous specification is also a
failure. Single-service selection uses an exact name within the selected stack.
Whole-stack rollback still requires an explicit confirmation in the script.

`SWARM_ROLLBACK_TIMEOUT_SECONDS` defaults to 300 seconds per service. After a
timeout, inspect Docker's task state before retrying because the daemon may still
be applying the requested rollback.
