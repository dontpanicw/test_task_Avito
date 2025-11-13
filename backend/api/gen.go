package api

//go:generate oapi-codegen -generate types  -package gen api/openapi.yaml > internal/input/http/gen/types.go
//go:generate oapi-codegen -generate chi-server,strict-server  -package gen api/openapi.yaml > internal/input/http/gen/server.go
//go:generate oapi-codegen -generate spec  -package gen api/openapi.yaml > internal/input/http/gen/spec.go

//go:generate oapi-codegen -generate types  -package client api/openapi.yaml > pkg/client/http/types.go
//go:generate oapi-codegen -generate client  -package client api/openapi.yaml > pkg/client/http/http_client.go
