module github.com/antithesishq/aardvark-arena

go 1.25.0

require github.com/google/uuid v1.6.0

require github.com/coder/websocket v1.8.14

require (
	github.com/antithesishq/antithesis-sdk-go v0.7.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/mattn/go-sqlite3 v1.14.33
)

require (
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/tools v0.42.0 // indirect
	hegel.dev/go/hegel v0.1.2 // indirect
)

tool github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor
