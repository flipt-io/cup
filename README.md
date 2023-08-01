> ⚠️ This is an active experiment into the benefits of managing an API over Git.
> Expect it to change quite frequently.
> Regardless, we really want your input, so please do give it a go.

`cup` An Instant API for Git
----------------------------

<div align="center">
  <img src="https://github.com/flipt-io/cup/assets/1253326/602edfd0-8da3-4b37-856a-8b620af0d264" alt="CUP" width="240" />
</div>

`cup` brings Git repositories to life

## Features

- 🔋 Materialize API resources directly from Git
- 🏭 Manage change through a declarative API
- 🔩 Extend `cup` in your language of choice

## Roadmap

- [ ] 🛰️ Track open proposals directly in `cup`
- [ ] 🔒 Secure access via authorization policies

## Dependencies

- Go (>= 1.20)
- An SCM (Currently supported: GitHub, Gitea)

## Building

`cup` is actively being developed and very much in its infancy.
For now, to play with `cup` you will need to clone this project and build from source.

### `cupd` Server

`cupd` is the server portion of the `cup` project.
It handles sources to target repositories, manifesting resource APIs and handling transformations through resource controllers.

```
go install ./cmd/cupd/...
```

#### Configuration

A `cupd` instance is configured via a `cupd.json` configuration file.

<details>

<summary>Configuring cup to manage Flipt resources</summary>

The following contains an example configuration for exposing [Flipt](https://flipt.io) feature flag state via `cup`.

The configuration exposes the two primary top-level Flipt resources:

- Flags
- Segments

The WASM runtime can be built using `gotip` (requires Go 1.21+) against the Flipt controller in this project:

```bash
cd ext/controllers/flipt.io

GOOS=wasip1 GOARCH=wasm gotip build -o v1alpha1/flipt.wasm ./v1alpha1/cmd/flipt/*.go
```

`cupd.json` configuration contents:

```json
{
  "api": {
    "address": ":8181",
    "source": {
      "type": "git",
      "git": {
        "url": "http://username:PAT@github.com/yourrepo/something.git",
        "scm": "github"
      }
    },
    "resources": {
      "flipt.io/v1alpha1/flags": {
        "controller": "flipt"
      },
      "flipt.io/v1alpha1/segments": {
        "controller": "flipt"
      }
    }
  },
  "controllers": {
    "flipt": {
      "type": "wasm",
      "wasm": {
        "executable": "ext/controllers/flipt.io/v1alpha1/flipt.wasm"
      }
    }
  },
  "definitions": [
    {
      "inline": {
        "apiVersion": "cup.flipt.io/v1alpha1",
        "kind": "ResourceDefinition",
        "metadata": {
          "name": "flags.flipt.io"
        },
        "names": {
          "kind": "Flag",
          "singular": "flag",
          "plural": "flags"
        },
        "spec": {
          "group": "flipt.io",
          "versions": {
            "v1alpha1": {
              "type": "object",
              "properties": {
                "key": { "type": "string" },
                "name": { "type": "string" },
                "type": { "enum": ["", "FLAG_TYPE_VARIANT", "FLAG_TYPE_BOOLEAN"] },
                "enabled": { "type": "boolean" },
                "description": { "type": "string" },
                "variants": {
                  "type": ["array", "null"],
                  "items": {
                    "type": "object",
                    "properties": {
                      "key": { "type": "string" },
                      "description": { "type": "string" },
                      "attachment": {
                        "type": "object",
                        "additionalProperties": true
                      }
                    }
                  }
                },
                "rules": {
                  "type": ["array", "null"],
                  "items": {
                    "type": "object"
                  }
                },
                "rollouts": {
                  "type": ["array", "null"],
                  "items": {
                    "type": "object"
                  }
                }
              },
              "additionalProperties": false
            }
          }
        }
      }
    },
    {
      "inline": {
        "apiVersion": "cup.flipt.io/v1alpha1",
        "kind": "ResourceDefinition",
        "metadata": {
          "name": "segments.flipt.io"
        },
        "names": {
          "kind": "Segment",
          "singular": "segment",
          "plural": "segments"
        },
        "spec": {
          "group": "flipt.io",
          "versions": {
            "v1alpha1": {
              "type": "object",
              "properties": {
                "enabled": { "type": "boolean" }
              },
              "additionalProperties": false
            }
          }
        }
      }
    }
  ]
}
```

</details>

### `cup` CLI

`cup` is a CLI which is heavily influenced by `kubectl`.
It can be used locally to interact and introspect into a running `cupd`.

```
go install ./cmd/cup/...
```

#### Usage

```bash
NAME:
   cup - a resource API for Git

USAGE:
   cup [global options] command [command options] [arguments...]

COMMANDS:
   config, c  Access the local configuration for the cup CLI.
   help, h    Shows a list of commands or help for one command
   discovery:
     definitions, defs  List the available resource definitions for a target source
   resource:
     get    Get one or more resources
     apply  Put a resource from file on stdin

GLOBAL OPTIONS:
   --config value, -c value  (default: "$HOME/Library/Application Support/cup/config.json")
   --output value, -o value  (default: "table")
   --help, -h                show help
```

## Appreciation

`cup` is built on the shoulders of giants and inspired by many awesome projects that came before.

Built on:

- The [Go](https://go.dev/) programming language
- The [Wazero](https://github.com/tetratelabs/wazero/) Go WASM runtime
- The wonderful SCMs (Gitea, GitHub etc.)

Inspired by:

- [Kubernetes](https://kubernetes.io/)
- Our own wonderful [Flipt](https://github.com/flipt-io/flipt).
