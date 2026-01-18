.PHONY: swagger
swagger:
	swag init -g cmd/meerkat/main.go -o docs

.PHONY: swagger-install
swagger-install:
	go install github.com/swaggo/swag/cmd/swag@latest

