module customer-support

go 1.24.3

replace github.com/ryanhill4L/agents-sdk => ../..

require (
	github.com/anthropics/anthropic-sdk-go v0.5.0
	github.com/ryanhill4L/agents-sdk v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.24 // indirect
	golang.org/x/sync v0.10.0 // indirect
)