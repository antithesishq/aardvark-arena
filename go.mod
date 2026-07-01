module github.com/antithesishq/aardvark-arena

go 1.25.0

require github.com/google/uuid v1.6.0

require github.com/coder/websocket v1.8.15

require (
	github.com/antithesishq/antithesis-sdk-go v0.7.2
	github.com/jmoiron/sqlx v1.4.0
	github.com/mattn/go-sqlite3 v1.14.47
	hegel.dev/go/hegel v0.6.7
)

require (
	github.com/ebitengine/purego v0.10.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
)

tool github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor
