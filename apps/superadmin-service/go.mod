module superadmin-service

go 1.25

require (
	api v0.0.0
	connectrpc.com/connect v1.19.1
	github.com/joho/godotenv v1.5.1
	github.com/obiente/cloud/apps/shared v0.0.0
	golang.org/x/net v0.43.0
)

replace api => ../api

replace github.com/obiente/cloud/apps/shared => ../shared
