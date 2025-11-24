module notifications-service

go 1.25

require (
	connectrpc.com/connect v1.19.1
	github.com/joho/godotenv v1.5.1
	github.com/obiente/cloud/apps/shared v0.0.0
	golang.org/x/net v0.43.0
	google.golang.org/protobuf v1.36.9
	gorm.io/gorm v1.31.0
)

replace github.com/obiente/cloud/apps/shared => ../shared

