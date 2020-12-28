module github.com/gherynos/vault-backend

go 1.15

replace github.com/gherynos/vault-backend/server => ./server

require (
	github.com/hashicorp/vault/api v1.0.4
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.3.0
)
