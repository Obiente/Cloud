module dns-service

go 1.25

require (
	api v0.0.0
	github.com/joho/godotenv v1.5.1
	github.com/miekg/dns v1.1.68
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.0
)

replace api => ../api
