module gameservers-service

go 1.25

require (
	connectrpc.com/connect v1.19.1
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/joho/godotenv v1.5.1
	github.com/moby/moby/api v1.52.0
	github.com/moby/moby/client v0.2.1
	github.com/obiente/cloud/apps/shared v0.0.0
	golang.org/x/net v0.43.0
	google.golang.org/protobuf v1.36.9
	nhooyr.io/websocket v1.8.17
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/redis/go-redis/v9 v9.16.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
	gorm.io/gorm v1.31.0 // indirect
)

exclude github.com/moby/moby v28.3.1+incompatible

exclude github.com/moby/moby v28.3.2+incompatible

exclude github.com/moby/moby v28.3.3+incompatible

exclude github.com/moby/moby v28.4.0+incompatible

exclude github.com/moby/moby v28.5.0+incompatible

exclude github.com/moby/moby v28.5.1+incompatible

exclude github.com/moby/moby v28.5.2+incompatible

exclude github.com/docker/docker v28.3.1+incompatible

exclude github.com/docker/docker v28.3.2+incompatible

exclude github.com/docker/docker v28.3.3+incompatible

exclude github.com/docker/docker v28.4.0+incompatible

exclude github.com/docker/docker v28.5.0+incompatible

exclude github.com/docker/docker v28.5.1+incompatible

exclude github.com/docker/docker v28.5.2+incompatible

replace github.com/obiente/cloud/apps/shared => ../shared
