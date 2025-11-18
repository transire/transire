module github.com/transire-org/simple-api

go 1.21

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/transire/transire v0.1.0
)

replace github.com/transire/transire => ../../

require (
	github.com/aws/aws-lambda-go v1.46.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
