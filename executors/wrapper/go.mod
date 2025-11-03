module github.com/hadi77ir/go-query/executors/wrapper/v2

go 1.24.0

require (
	github.com/hadi77ir/go-query v1.2.0
	github.com/hadi77ir/go-query/executors/memory v1.2.0
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/hadi77ir/go-query => ../..

replace github.com/hadi77ir/go-query/executors/memory => ../memory
