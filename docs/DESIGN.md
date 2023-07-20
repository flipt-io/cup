Design
------

This document outlines an initial MVP design for a `cup` alpha release.
It aims to outline some initial decisions around how `cup` structures it's internal interactions and external API.

## Intentions

We've taken a heavy dose of inspiration for the API from Kubernetes.
The object and resource model is a good potential fit for `cup`, so we want to replicate it as much as possible.
The self-describing API should lend itself well to building additional tooling (CLI, UI, etc.).

While we're taking a lot of inspiration from it, we're don't want to prescribe this tool to a Kubernetes only audience.
We feel it has scope to be valuable beyond the Kubernetes ecosystem.

As such, there will be a bit of wheel re-invention at the start of this project.
We don't want folks to have to have a Kubernetes cluster, or for `cup` to have a hard dependency on all the Kubernetes internals.
We will definitely pick and pluck in dependencies as and where they make sense (to speed things up or ensure consistency where it makes sense).

## Goals

Enable folks to put declarative APIs onto their Git repositories in order to gradually build automation and internal tooling in and around their configuration repositories.

## Non-Goals

This is **not** an attempt to build a compatible Kubernetes API server backed by Git.
We purely see there being value in the approach taken by Kubernetes to represent resources internally and via the API.

## Overview

The goal is to provide users with a way to manage the logical resources represented in Git repositories.
`cup` takes care of extracting resources and translating your desired changes into commits, branches and pull-requests.

```
┌─────┐            ┌──────────────────────────────┐         ┌──────────────────────────────────┐
│     │            │Executor                      │         │ GitHub                           │
│     │            │                              │         │                                  │
│     │            │  flipt.io/Flag/v1            │         │  ┌─────────────────────┐         │
│     ├────────────▶                              │         │  │Git Repository       │         │
│     │            │ ┌──────────────────────────┐ │         │  │                     │         │
│     │            │ │Controller                │ │  ┌──────┼─▶│                     │         │
│     │            │ │                          │ │  │      │  │                     │         │
│     │            │ │ exec type                │ │  │      │  │                     │         │
│     │            │ │ exec get ...             │ │  │      │  └─────────────────────┘         │
│     │            │ │ exec list ...            │ │  │      │  ┌─────────────────────┐         │
│     │            │ │ exec put ...             │ │  │      │  │Git Repository       │         │
│     │            │ │ exec delete ...          │ │  │      │  │                     │         │
│     │            │ │                          │ │  │      │  │                     │         │
│     │            │ └───────▲─────────┬────────┘ │  │      │  │                     │         │
│  A  ◀────────────┤         │         │          │  │      │  │                     │         │
│  P  │            │ ┌───────┴─────────▼────────┐ │  │      │  └─────────────────────┘         │
│  I  │            │ │Filesystem                ◀─┼──┘  ┌───▶                                  │
│     │            │ │                          │ │     │   │                                  │
│  S  │            │ └──────────────────────────┘ ├─────┘   └──────────────────────────────────┘
│  e  │            └──────────────────────────────┘         ┌──────────────────────────────────┐
│  r  │                                                     │ GitLab                           │
│  v  │                                                     │                                  │
│  e  │            ┌──────────────────────────────┐         │  ┌─────────────────────┐         │
│  r  │            │Executor                      │         │  │Git Repository       │         │
│     ├────────────▶                              │         │  │                     │         │
│     │            │  apps/Deployment/v1          │         │  │                     │         │
│     ◀────────────┤                              │         │  │                     │         │
│     │            │ ...                          │         │  │                     │         │
│     │            └──────────────────────────────┘         │  └─────────────────────┘         │
│     │            ┌──────────────────────────────┐         │  ┌─────────────────────┐         │
│     │            │Executor                      │         │  │Git Repository       │         │
│     ├────────────▶                              │         │  │                     │         │
│     │            │  my.org/Server/v1alpha1      │         │  │                     │         │
│     ◀────────────┤                              │         │  │                     │         │
│     │            │ ...                          │         │  │                     │         │
│     │            └──────────────────────────────┘         │  └─────────────────────┘         │
│     │                                                     │                                  │
│     │              ...                                    │                                  │
└─────┘                                                     └──────────────────────────────────┘
```

## Resources

Resources are the extension point of `cup`.
They are the interface chosen by `cup` operators and configurers to expose over a chosen Git repository.
A resource can be whatever you want to store or represent in your Git repository.

For example:
- Service configuration (e.g. source artefacts, log levels, resource quotas and so on)
- Feature flag state
- CI/CD Pipelines

Resources are derived directly from Kubernetes Resources.

A Resource is typed using the three following fields:

- Group (a collection of related kinds)
- Version
- Kind (a single type name)

Each unique instance of a resource is identified by a `namespace` and `name`.

A resource can also have additional metadata in the form of annotations and labels.
Labels is a map of key/value string pairs, which is then indexed and facetable when listing resource types via the API.
Annotations is a map of arbitrary key/value string pairs which is not indexed, but intended for external tooling to leverage for their purposes.

Each resource contains a specification field, which is unique per resource type.
The contents of this field is controlled via a schema defined on the Resource Definition

**Example Resource Payload**

```json
{
  "apiVersion": "flip.io/v1alpha1",
  "kind": "Flag",
  "metadata": {
    "namespace": "production",
    "name": "new-project-cup",
    "labels": {
      "project-type": "moonshots"
    },
    "annotations": {}
  }
  "spec": {}
}
```

### Resource Definition

A resource definition is similarly structured to that of a Resource itself.
It is inspired by, but differs in slight ways to that of Kubernetes Custom Resource Definitions.
It's purpose is to identify a type and associated schema for validating the specification section of an individual resource.
The schema provided is defined using JSONSchema syntax and pertains to what can go in the `spec` field of a particular resource.

Schemas can be defined per version of the group and kind.

The core specification includes a section which defines where and how to source the Controllers implementation.
Currently, this `controller` field has a single field `path`.
This is a relative or absolute path to a target WASM/WASIP1 binary implementation of a Controller.

> Depending on how the resource definition is sourced effects the meaning of the relative path.
> If the definition is inline within the server configuration, then it is relative to the server processes current working directory.
> If the definition itself is sourced from a path on disk, then the path will be relative to the same directory as the resource definition.
> In a potential future where `cup` controller are packaed into OCI images, this path would be relative to inside the OCI artefact itself.

**Example Resource Definition Payload**

```json
{
  "apiVersion": "cup.flipt.io/v1alpha1",
  "kind": "ResourceDefinition",
  "metadata": {
    "name": "flags.flipt.io",
  }
  "names": {
    "kind": "Flag",
    "singular": "flag",
    "plural": "flags",
  },
  "spec": {
    "group": "flipt.io",
    "controller": {
      "path": "flipt.wasm",
    },
    "versions": {
      "v1": {
        "schema": {
          "type": "object"
          "properties": {
            "enabled": {"type": "boolean"}
            "description": {"type": "string"}
          }
        }
      }
    }
  }
}
```

## API Server

`cup` is fronted with an API that can be used to discover which resource types are available, as well as read and write resource instances themselves.

The API will initially have two make categories of functionality:

1. Resource type cataloging
2. Resource type instance management

### Resource Type Catalog

This section of the API is focussed on supporting callers discovering which types are registered for consumption.

### Resource Type Instance APIs

This section of the API is manifested based on the registered resource types for the configured Git repositories.

The root prefix for all of these APIs will be `/apis`.

Each resource types will have its own relevant prefix of the API surface area, which will support:

- Getting individual resources
- Listing (and filtering) sets of the given resource type
- Putting the state of individual resources
- Deleting individual resources

Each API section will be prefixed in the form: `/apis/<group>/<version>/<plural>`.

The following is an explanation of each of the path parameters:
- `group` refers to the resource type group
- `version` refers to the resouece type version
- `plural` refers to the resource type `names.plural` value

When a request is made for a particular type, the API Server parses, validates and then delegates the request onto a relevant `Executor`.

### Authentication and Authorization

This section is TBD.

Given we're taking inspiration from the Kubernetes resource and API structure we've co-opted, the intent is we can leverage the resource metadata (types, namespaces, names) and operation verbage (get, list, put and delete) in some kind of future authorization policy language.
Likelihood is we explore something close to the RBAC mechanisms available in the Kubernetes ecosystem.

## Executor

Executors sit at the heart of `cup`. A single executor handles processing requests for a single resource type via it's associated controller.

An executor has a general behaviour over any given `Controller` implementation.
`Controller` implementations are exposed through a process command-line interface.
Each controller binary is compiled to WASM, available on the local filesystem.

The executor will take care of adapting each request into an appropriate set of command line arguments and/or STDIN written payloads.
It then interprets any exist codes and output written to the standard output streams (STDOUT / STDERR).

It is also the executors job to prepare the WASM runtime environment for a given request.
A request should identify either a specific Git SHA or reference.
The executor will retrieve a read-only snapshot of the entire Git tree for the resolved revision.
This will be mounted as the root filesystem for the WASM runtime.

Given a mutating operation is request (`put` or `delete`) then the executor will support writes on the filesystem.
The executor will intercept these writes and ultimately compose them into a pull request containing the changes made.

## Controllers

### get

Retrieving an instance of a resource by `namespace` and `name`.

```
exec wasm ["get", "<namespace>", "<name>"]                     
        ┌──────────────────────┐                               
        │                      │        {                      
        │     WASM Binary      │            "apiVersion": "..."
        │                      │            "kind": "...",     
        │                      │            ...                
        │                      ├──────▶ }                      
        └──────────────────────┘                               
```

The purpose of this subcommand is to address an instance by namespace and name.
It should handle the sub-command `get`.
Then the following two arguments will the `namespace`, followed by the `name` of the instance.

The resource should be extracted from the local-filesystem.
The filesystem will contain the configured target Git repositories HEAD tree for the resolved reference mounted at `/`.

#### Output

| Meaning   | Exit code | STDOUT                |
| --------- | --------- | --------------------- |
| success   | 0         | JSON encoded resource |
| error     | 1         | JSON encoded message  |
| not found | 2         | JSON encoded message  |

### list

Listing and filtering a set of resource instances by `namespace` and optional `labels`

```
exec wasm ["list", "<namespace>", ...(k/v pairs)]                  
        ┌──────────────────────┐                                   
        │                      │        [{                         
        │     WASM Binary      │            "apiVersion": "..."    
        │                      │            "kind": "...",         
        │                      │            ...                    
        │                      ├──────▶ }, ...]                    
        └──────────────────────┘                                   
```

The purpose of this subcommand is to return a list of instances found by the target controller.
The controller should handle filtering by namespace and optionall by a list of `key=value` pairs of labels.

#### Output

| Meaning   | Exit code | STDOUT                       |
| --------- | --------- | ---------------------------- |
| success   | 0         | JSON encoded resource stream |
| error     | 1         | JSON encoded message         |
| not found | 2         | JSON encoded message         |

### put

Creating or updating an existing resource.

```
exec wasm ["put"]                              
                              ┌──────────────────────┐               
{                             │                      │               
    "apiVersion": "..."       │     WASM Binary      │               
    "kind": "...",            │                      │               
    ...                       │                      │               
}                       ──────▶                      ├──────▶ { TBD }
                              └──────────────────────┘               
```

The purpose of this subcommand is to create a new or update (upsert) an existing resource.
Implementations should adjust the filesystem appropriately for the resource type and controllers needs.
The new resource payload is serialized on STDIN.

TBD:

- What makes sense to return from the binary?

#### Output

| Meaning   | Exit code | STDOUT               |
| --------- | --------- | -------------------- |
| success   | 0         | TBD                  |
| error     | 1         | JSON encoded message |

#### Flow

This diagram gives an overview of the flow of a successsful `PUT` request:

```mermaid
sequenceDiagram
    participant A as Actor
    participant S as API Server
    participant E as Executor
    participant F as Worktree
    participant R as Controller
    participant G as Git
    participant SCM
    A ->> S: PUT /apis/g/v/k
    S ->>+ E: Put(Resource{})
    E ->>+ G: checkout()
    G ->>- F: tree
    E ->>+ R: exec wasm put { ... }
    R ->> F: write()
    R ->>- E: exit 0
    E ->> F: git add
    E ->> G: commit and push
    E ->>+ SCM: OpenPR()
    SCM ->>- E: PR{}
    E ->>- S: Status{}
    S ->> A: 202 Accepted
```

### delete

Removing an existing resource.

```
exec wasm ["delete", "<namespace>", "<name>"]                  
        ┌──────────────────────┐                               
        │                      │                               
        │     WASM Binary      │                               
        │                      │                               
        │                      │                               
        │                      ├──────▶ { TBD }                
        └──────────────────────┘                               
```

The purpose of this subcommand is to remove an existing resource.
Implementations should adjust the filesystem appropriately for the resource type and controllers needs.
The namespace and name of the resource is passed as arguments to the subcommand.

TBD:

- What makes sense to return from the binary?

## SCM and Git providers

Proposed implementations:

- GitHub
- GitLab
- Gitea
