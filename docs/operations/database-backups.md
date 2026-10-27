# Managed database backups

Managed backups store compressed SQL exports of the instance's primary database
on an operator-provided shared volume. PostgreSQL, MySQL and MariaDB are supported.
Backups are created on demand. There is no automatic schedule. Additional databases
created inside an instance and server accounts are not included.

## Configure shared storage

Use durable storage independent of database containers. For a Swarm deployment,
the volume must expose the same files on every node eligible to run the database
control plane. A local Docker volume on each node does not provide shared storage.
Keep access limited to the database service, use encrypted storage where required,
and retain an independent off-host copy for disaster recovery.

1. Provision and mount the shared volume using your storage driver's instructions.
2. At the root of that volume, create a file named `.obiente-backup-volume` whose
   exact contents are `obiente-cloud-backups-v1` followed by a newline. The service
   deliberately never creates this marker or the root directory. A missing marker
   disables backup operations instead of writing into an unmounted directory.
3. Give the service account access to the directory. Backup files are written with
   mode `0600`; the filesystem must support atomic rename and file/directory sync.
4. Set `DATABASE_BACKUP_VOLUME` to the existing volume's Docker name. Include
   `docker-compose.backups.yml` after `docker-compose.yml` or
   `docker-compose.swarm.yml` in your normal Compose/stack deployment command.
   For custom stacks, apply the same mount and environment to the database
   control-plane service, not to proxy-only tasks.

| Setting                     | Default      | Meaning                                                            |
| --------------------------- | ------------ | ------------------------------------------------------------------ |
| `DATABASE_BACKUP_DIRECTORY` | unset        | Existing mounted directory inside the service; the overlay sets it |
| `DATABASE_BACKUP_KEEP`      | `10`         | Completed backups retained per database, from 1 to 1000            |
| `DATABASE_BACKUP_MAX_BYTES` | `1073741824` | Maximum uncompressed dump size per backup and restore              |

The dashboard reports volume availability and the retention count. A successful
mount alone is insufficient: validate create and restore against a disposable
database before relying on the storage. The database service also needs its normal
Docker access and database credential decryption key after a restart.

## Create and restore

Open a database's Backups tab and select **Create backup**. The initial state is
Creating. The page refreshes automatically and reports Ready only after the dump,
compression, checksum, atomic file publication and metadata update finish.
The worker processes durable requests in the metadata database and serializes
operations with database lifecycle commands across control-plane replicas.

**Restore** requires a new database name. The service checks the complete backup
file and checksum before creating the target. Existing databases are never
overwritten. Once the restore completes, use the new database name with the
instance's existing connection credentials. Validate the restored application
data before switching clients to it.

Creation requires `database.update`, restoration requires `database.manage`,
reading requires `database.read`, and deletion requires `database.delete`.
Queued create/restore work rechecks the requesting user's current access.

## Failure and recovery

- Incomplete exports can be retried from durable work after an abrupt restart.
  Partial files are never advertised as completed backups.
- A stopped or unavailable database produces a failed operation. Backups do not
  override an operator's stop intent.
- Imports interrupted after they start are marked failed after restart and are
  never automatically repeated. The new target may contain partial data. Inspect
  it and use a different target name when retrying.
- Deletions and retention cleanup resume after restart. Retention removes older
  completed backups only after a new backup has completed successfully.
- Checksum failures, missing files, unavailable volumes, size limits and expired
  operation deadlines are failures, never successful empty backups.

Retain the metadata database together with the backup volume. Metadata contains
ownership, checksums and engine information required for managed restoration.
Compressed files contain ordinary SQL and can also be recovered with the matching
engine's native tools by an authorized operator. Do not edit backup files in place.
