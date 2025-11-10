module vps-gateway

go 1.25

require (
	connectrpc.com/connect v1.19.1
	github.com/obiente/cloud/apps/shared v0.0.0
	github.com/prometheus/client_golang v1.20.5
	github.com/prometheus/common v0.55.0
	google.golang.org/protobuf v1.36.9
)

replace github.com/obiente/cloud/apps/shared => ../shared

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	golang.org/x/sys v0.37.0 // indirect
)
