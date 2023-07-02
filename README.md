cup - GitOps Contribution Automation Tool
-----------------------------------------

`cup` brings Git repositories to life.

Instantly expose the contents of your Git repositories via the `cup` API.
Get, List, Update and Delete the logical resources represented within them.
Let `cup` handle pushing contributions and opening pull-requests when using the mutation APIs.

Use common pre-built runtimes, referred to as `sourcers`, to expose common configuration formats by simply configuring `cup`.
Alternatively, build your own `sourcers` in any language which compiles to WASM WASI preview1.

## Building

### Requirements

- Go

## Running

```sh
# serve current working directory
go run cmd/cup/main.go

# serve an upstream repository
go run cmd/cup/main.go -source git -repository http://flipt:password@localhost:3001/flipt/features.git
```

## Sourcers

`cup` requires `sourcers` to collect and transform the logical resources described within your Git repositories.

A `sourcer` is a WASM binary compiled for a WASI execution environment (specifically `cup` uses Wazero).
Each binary should conform to the same API. Currently, that consists of four sub-commands.

The sub-commands should parse the local filesystem for state you want to expose.
Each `sourcer` represents a single logical resource encoded in your repository.
For example, Flipt's feature flags can be encoded in a repository via a YAML configuration format.

Checkout the [Flipt Flag](./cmd/flipt/main.go) implementation for an example.
This uses the [sdk/go](./sdk/go) package to simplify bootstrapping a `sourcer` implementation.
It allows developers to focus on the business logic of finding resources and updating them.
You simply need to implement the `sdk.TypedRuntime[T]` generic interface.

### CLI Commands

Each of the following should be handled as the first argument after the program name:

- `type`

Should return a JSON encoded object containing the group, kind and version of the `sourcer` instance.

```sh
➜  go run cmd/flipt/main.go type | jq
{
  "group": "flipt.io",
  "kind": "Flag",
  "version": "v1"
}
```

- `list`

Should return a stream of JSON encoded resources on STDOUT (JSON-LD stream).
The entire contents is expected to be listed when this is called.
Each resource should expose a `namespace` and an `id`.
Each `id` must be unique within each `namespace`.

```sh
➜  go run cmd/flipt/main.go list
{"namespace":"default","id":"flag1","payload":{"name":"flag1","description":"description","enabled":true,"variants":[...],"rules":[...]}}
{"namespace":"default","id":"foo","payload":{"name":"Foo","description":"","enabled":false,"variants":null,"rules":null}}
```

- `put`

Takes a single JSON encoded resource in STDIN to be created or updated (upsert).
The implementation should locate all files which relate to the resource and re-render them appropriately.
Each files new contents must be streamed back out as a JSON encoded object with the path, contents and a message describing the change.

```
➜  go run cmd/flipt/main.go put <<EOF | jq -r
{"namespace":"default","id":"foo","payload":{"name":"Foo","enabled":true}}
EOF
{
  "path": "features.yml",
  "message": "feat: update flag \"default/foo\"",
  "contents": "bmFtZXNwYWNlOi..."
}
```

- `delete`

Takes a namespace and id for a particular resource as command-line arguments.
The implementation should locate all files related and remove the related resource data.
Once again, the new contents of each file should be streamed as JSON encoded objects with the path, new contents and message describing the change.

```sh
➜  go run cmd/flipt/main.go delete default foo
{"path":"features.yml","message":"feat: delete flag \"default/foo\"","contents":"bmFtZXNw..."}
```
