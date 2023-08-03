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

A `cupd` instance has a number of configuration mechanisms.
It first helps to understand the different mechanisms involved in configuring a Cup server.

1. General top-level `cupd` configuration
2. Resource definitions
3. Controller configuration
4. API resource / controller bindings

##### General Configuration

This is the top-level set of configuration which can be provided via CLI flags, environment variables or a configuration yaml file.
You can see what `cupd` requires by simple invoking the following:
```
➜  cupd serve -h
DESCRIPTION
  Run the cupd server

USAGE
  cupd serve [flags]

FLAGS
  -api-address :8181    server listen address
  -api-git-repo string  target git repository URL
  -api-git-scm github   SCM type (one of [github, gitea])
  -api-local-path .     path to local source directory
  -api-resources .      path to server configuration directory (controllers, definitions and bindings)
  -api-source local     source type (one of [local, git])
```

One thing of note is the `-api-resources` flag which configures the location of `cupd`'s API resource directory.
This directory should contain a bunch of API resource instances which we will talk about in the following configuration sections.

Each of the flags can be altneratively provided as an environment variable.
The convention for environment variable naming is: `CUPD{{ uppercase(replace(flag, "-", "_")) }}`.

For example, `-api-address` can be expressed via `CUPD_API_ADDRESS`.

Finally, a configuration YAML file can be used instead.
This also follows a naming convention which tokenizes flag keys on `-`.

<details>

<summary>Example cup config.yml</summary>

```yaml
api:
  address: ":8181"
  source: "git"
  resources: "/etc/cupd/config/resources"
  git:
    scm: "github"
    repo: "http://username:PAT@github.com/yourrepo/something.git"
```

</details>

##### Resource Definitions

Resource definitions live in the directory identified by `-api-resources`.
Each definition contains the group, kind and versioned schemas for resource types handled by Cup.
These definitions are heavily inspired by Kubernetes concept of Customer Resource Definitions.

Any file in the API resources directory ending in `.json` is currently parsed and interpretted.
Depending on the `apiVersion` and `kind` of the resource, they each get treated accordingly.

Each resource definition configuration payload include the following top-level fields:

| Key        | Value                      |
|------------|----------------------------|
| apiVersion | `"cup.flipt.io/v1alpha1"`  |
| kind       | `"ResourceDefinition"`     |
| metadata   | `<Metadata>`               |
| names      | `<Names>`                  |
| spec       | `<ResourceDefinitionSpec>` |

<details>

<summary>Example Flipt flag resource definition</summary>

```json
{
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
