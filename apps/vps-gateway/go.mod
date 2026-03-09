module vps-gateway

go 1.25

require (
	connectrpc.com/connect v1.19.1
	github.com/obiente/cloud/apps/shared v0.0.0
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/common v0.66.1
	github.com/redis/go-redis/v9 v9.16.0
	golang.org/x/net v0.47.0
	google.golang.org/protobuf v1.36.10
)

replace github.com/obiente/cloud/apps/shared => ../shared

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
)
