<div align="center">
  <a href="https://github.com/flipt-io/cup/blob/main/go.mod">
    <img alt="Go version 1.21" src="https://img.shields.io/github/go-mod/go-version/flipt-io/cup">
  </a>
  <a href="https://github.com/flipt-io/cup/blob/main/LICENSE">
    <img alt="Apache v2" src="https://img.shields.io/github/license/flipt-io/cup">
  </a>
  <a href="https://github.com/flipt-io/cup/actions">
      <img src="https://github.com/flipt-io/cup/actions/workflows/test.yml/badge.svg" alt="Build Status" />
  </a>
  <a href="https://discord.gg/kRhEqG2TEZ">
      <img alt="Discord" src="https://img.shields.io/discord/960634591000014878?color=%238440f1&label=Discord&logo=discord&logoColor=%238440f1&style=flat">
  </a>
</div>

> ⚠️ This is an active experiment into the benefits of managing an API over Git.
> Expect it to change quite frequently.
> Regardless, we really want your input, so please do give it a go.

`cup` An Instant API for Git
----------------------------

<div align="center">
  <img src="https://github.com/flipt-io/cup/assets/1253326/d408dbe2-51bf-414e-93ec-603e09d5c1fa" alt="CUP" width="240" />
</div>

Cup helps you build APIs and automation ontop of your Git repositories.

A configurable and extensible server for managing and exposing API resources directly from a target Git repository.
It exposes a Kubernetes-like declarative API, which organizes resources into typed (group + version + kind) sets.
Resources can be listed, read, updated, and deleted. When changes to the state of a resource are made, the resulting
calculated difference is automatically proposed as a pull or merge request on a target Git SCM.
How resources map to and from API request payloads to files in your repository is handled by [Controllers](#controllers).
Controllers are configurable and broadly extensible through the power of WASM via the [Wazero](htts://github.com/tetratelabs/wazero) runtime.

## 📣 Feedback

We really want to learn how you do configuration management.
If you have a second, we would greatly appreciate your input on this [feedback form](https://1ld82idjvlr.typeform.com/to/egIn3GLO).

[Cup and Flipt Demo Video](https://github.com/flipt-io/cup/assets/1253326/9c045493-c7c1-44ad-9066-9649de8b57c1)

## Table of Contents

- [Features](#features)
- [Roadmap](#roadmap)
- [Use-cases](#use-cases)
- [Dependencies](#dependencies)
- [Server](#server)
- [CLI](#cli)

## Features

![cup-diagram](https://github.com/flipt-io/cup/assets/1253326/7a88d16c-c2c9-4d5b-8547-02c71043fd27)

- 🔋 Materialize API resources directly from Git
- 🏭 Manage change through a declarative API
- 🔩 Extend using the power of WASM

## Roadmap

- [ ] 📦 Package and distribute controllers as OCI images
- [ ] 🛰️ Track open proposals directly through the `cupd` API
- [ ] 🔒 Secure access via authorization policies

## Use-cases

Cup is a foundation on which to build tooling around configuration repositories.
We imagine folks may find all sorts of weird and wonderful applications for Cup (and we want to hear about them).

Some ideas we're brewing:

- A central CLI for exploring and editing the state of your configuration repositories
- A dashboard for exploring and editing how your services are configured
- Access controlled management for infrastructure change requests
- New project or service templating (project structure, build, test and deploy pipelines)
- Expose configuration controls (e.g. feature flags, resource requests) to non-Git users

## Dependencies

- Go (>= 1.20)
- An SCM (Currently supported: GitHub, Gitea)

## Server

The server component of the Cup project is known as `cupd`.
It is a configurable API server, which exposes and manages the state of a target repository.

### Building

For now, to play with `cupd` you will need to clone this project and build from source.

From the root of this project, run:

```console
mkdir -p bin

go build -o bin/cupd ./cmd/cupd/...
```

This will produce a binary `cupd` in the local folder `bin`.

### Usage

```console
➜  cupd serve -h
DESCRIPTION
  Run the cupd server

USAGE
  cupd serve [flags]

FLAGS
  -api-address :8181          server listen address
  -api-git-repo string        target git repository URL
  -api-git-scm github         SCM type (one of [github, gitea])
  -api-local-path .           path to local source directory
  -api-resources .            path to server configuration directory (controllers, definitions and bindings)
  -api-source local           source type (one of [local, git])
  -tailscale-auth-key string  Tailscale auth key (optional)
  -tailscale-ephemeral=false  join the network as an ephemeral node (optional)
  -tailscale-hostname string  hostname to expose on Tailscale
```

## CLI

`cup` is a CLI that is heavily influenced by `kubectl`.
It can be used locally to interact and introspect a running `cupd`.

### Building

```console
mkdir -p bin

go build -o bin/cup ./cmd/cup/...
```

This will produce a binary `cup` in the local folder `bin`.

### Usage

```console
NAME:
   cup - Manage remote cupd instances

USAGE:
   cup [global options] command [command options] [arguments...]

COMMANDS:
   config, c  Access the local configuration for the cup CLI.
   help, h    Shows a list of commands or help for one command
   discovery:
     definitions, defs  List the available resource definitions
   resource:
     get     Get one or more resources
     apply   Put a resource from file on stdin
     edit    Edit a resource
     delete  Delete a resource

GLOBAL OPTIONS:
   --config value, -c value     (default: "/Users/georgemac/Library/Application Support/cup/config.json")
   --output value, -o value     (default: "table")
   --address value, -a value
   --namespace value, -n value
   --level value, -l value      set the logging level (default: "info")
   --help, -h                   show help
```

## Appreciation

`cup` is built on the shoulders of giants and inspired by many awesome projects that came before.

Built on:

- The [Go](https://go.dev/) programming language
- The [Wazero](https://github.com/tetratelabs/wazero/) Go WASM runtime
- The wonderful SCMs (Gitea, GitHub, etc.)

Inspired by:

- [Kubernetes](https://kubernetes.io/)
- Our own wonderful [Flipt](https://github.com/flipt-io/flipt)
