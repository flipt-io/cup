> ⚠️ This is an active experiment into the benefits of managing an API over Git.
> Expect it to change quite frequently.
> Regardless, we really want your input, so please do give it a go.

`cup` An Instant API for Git
----------------------------

<div align="center">
  <img src="https://github.com/flipt-io/cup/assets/1253326/602edfd0-8da3-4b37-856a-8b620af0d264" alt="CUP" width="240" />
</div>

Cup brings Git repositories to life.

A configurable and extensible server for managing and exposing API resources directly from a target Git repository.
It exposes a Kubernetes-like declarative API, which organizes resources into typed (group + version + kind) sets.
Resources can be listed, read, updated, and deleted. When changes to the state of a resource are made, the resulting
calculated difference is automatically proposed as a pull or merge request on a target Git SCM.
How resources map to and from API request payloads to files in your repository is handled by [Controllers](#controllers).
Controllers are configurable and broadly extensible through the power of WASM via the [Wazero](htts://github.com/tetratelabs/wazero) runtime.

## 📣 Feedback

We really want to learn how you do configuration management.
If you have a second, we would greatly appreciate your input on this [feedback form](https://1ld82idjvlr.typeform.com/to/egIn3GLO).

## Features

![cup-diagram](https://github.com/flipt-io/cup/assets/1253326/5494a487-796a-4462-a37e-3e4b0d01f5f1)

- 🔋 Materialize API resources directly from Git
- 🏭 Manage change through a declarative API
- 🔩 Extend `cup` in your language of choice

## Roadmap

- [ ] 🛰️ Track open proposals directly in `cup`
- [ ] 🔒 Secure access via authorization policies

## Table of Contents

- [Dependencies](#dependencies)
- [Building](#building)
  - [`cupd` Server](#cupd-server)
    - [Configuration](#configuration)
      - [General](#general-configuration)
      - [Resource Definitions](#resource-definitions)
      - [Controllers](#controllers)
      - [Bindings](#bindings)
  - [`cup` CLI](#cup-cli)

## Dependencies

- Go (>= 1.20)
- An SCM (Currently supported: GitHub, Gitea)

## Building

`cup` is actively being developed and very much in its infancy.
For now, to play with `cup` you will need to clone this project and build from source.

### `cupd` Server

`cupd` is the server portion of the `cup` project.
It handles sources to target repositories, manifesting resource APIs, and transformations through resource controllers.

```
go install ./cmd/cupd/...
```

#### Configuration

A `cupd` instance has a number of configuration mechanisms.
It first helps to understand the different mechanisms involved in configuring a Cup server.

1. General top-level `cupd` configuration
2. Resource Definitions
3. Controllers
4. Bindings

##### General Configuration

This is the top-level set of configuration which can be provided via CLI flags, environment variables, or a configuration YAML file.
You can see what `cupd` requires by simply invoking the following:
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

Each of the flags can be alternatively provided as an environment variable.
The convention for environment variable naming is: `CUPD{{ uppercase(replace(flag, "-", "_")) }}`.

For example, `-api-address` can be expressed via `CUPD_API_ADDRESS`.

Finally, a configuration YAML file can be used instead.
This also follows a naming convention that tokenizes flag keys on `-`.

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
These definitions are heavily inspired by Kubernetes' concept of Customer Resource Definitions.

Any file in the API resources directory ending in `.json` is currently parsed and interpreted.
Depending on the `apiVersion` and `kind` of the resource, they each get treated accordingly.

Each resource definition configuration payload includes the following top-level fields:

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

##### Controllers

Controllers are the engines that drive reading and writing changes to target configuration sources (directories, repositories etc.).
Internally, Cup abstracts away the details of Git repositories, commits, trees, and pull requests.
Controllers focus on handling individual actions over a mounted target directory.

These actions currently include:

- get
- list
- put
- delete

Each controller is configured via a JSON file in the `-api-resources` directory.
The files contain the following top-level fields:

| Key        | Value                     |
|------------|---------------------------|
| apiVersion | `"cup.flipt.io/v1alpha1"` |
| kind       | `"Controller"`            |
| metadata   | `<Metadata>`              |
| spec       | `<ControllerSpec>`        |

There are currently two types of controller configurable in Cup:

- Template
- WASM

**Template**

Is a simple, built-in controller which uses Go's `text/template` to perform 1:1 API resource to file in repo mappings.

There exist two templates:

1. Directory Template

This is used during `list` operations to map a namespace onto a target directory exclusively containing files that each represent a single resource instance.
It can be configured with glob syntax to further constrain, which files in a target directory are considered resources.

2. Path Template

This is used during `get`, `put`, and `delete` operations to identify the particular file the resource should be read from, written to, or deleted respectively.

<details>

<summary>Example template controller</summary>

```json
{
  "apiVersion": "cup.flipt.io/v1alpha1",
  "kind": "Controller",
  "metadata": {
    "name": "some-template-controller"
  },
  "spec": {
    "type": "template",
    "spec": {
      "directory_template": "{{ .Namespace }}/*.json"
      "path_template": "{{ .Namespace }}/{{ .Group }}-{{ .Version }}-{{ .Kind }}-{{ .Name }}.json"
    }
  }
}
```

</details>

**WASM**

The WASM controller is an extension point that opens Cup up to the full power of languages which can be compiled to WASM with the WASIP1 extensions.
Given your controller can be expressed as a command-line tool, conforming to Cup's well-defined set of sub-commands and standard I/O expectations, implemented in a language compiled to WASM with WASIP1, then it can be used in Cup.

To learn more about the binary API required for a Cup WASM controller, check out the [Design](./docs/DESIGN.md) document in this repo.
Cup is a very early prototype and this design is likely to change and open to suggestions.

Given your resulting controller WASM binary is present and reachable on the filesystem by `cupd`, then it can be leveraged by Cup to handle your particular resource management needs.

The Controller resource specification includes a single field `path` which should point to where the WASM implementation exists on disk.
This path will resolve relative to the `-api-resources` directory.

<details>

<summary>Example Flipt WASM controller</summary>

```json
{
  "apiVersion": "cup.flipt.io/v1alpha1",
  "kind": "Controller",
  "metadata": {
    "name": "flipt"
  },
  "spec": {
    "type": "wasm",
    "spec": {
      "path": "flipt.wasm"
    }
  }
}
```

</details>

##### Bindings

Bindings are the last crucial mechanism for exposing resources via the Cup API.
A binding defines which resource types should be exposed and what controller should handle their operations.

<details>

<summary>Example Binding for Flipt resources and associated controller</summary>

```json
{
  "apiVersion": "cup.flipt.io/v1alpha1",
  "kind": "Binding",
  "metadata": {
    "name": "flipt"
  },
  "spec": {
    "controller": "flipt",
    "resources": [
      "flipt.io/v1alpha1/flags",
      "flipt.io/v1alpha1/segments"
    ]
  }
}
```

</details>

### `cup` CLI

`cup` is a CLI that is heavily influenced by `kubectl`.
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
- The wonderful SCMs (Gitea, GitHub, etc.)

Inspired by:

- [Kubernetes](https://kubernetes.io/)
- Our own wonderful [Flipt](https://github.com/flipt-io/flipt).
