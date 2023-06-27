fidgit - GitOps Contribution Automation Tool
--------------------------------------------

`fidgit` brings configuration repositories to life.

At its core it is a specification for cataloging and transforming the contents of Git repositories.

## Today

Today `fidgit` can manage a single type of resource: Flipt Flags (not Segments).

### Example

#### Running `fidgit`

1. Checkout the labs project
2. `cd chatbot`
3. `docker-compose -f docker-compose.gitops.yml up`
4. Then run the following:

```sh
go run cmd/fidgit/main.go -source git -repository http://flipt:password@localhost:3001/flipt/features.git
```

From here, `fidgit` loads the one runtime it currently has configured: `flipt.io/Flag/v1alpha1`.

This collection runtime is mounted on the server at `/api/v1/flipt.io/Flag/v1apha1/`

#### Getting Flags

```sh
➜  curl --silent localhost:9191/api/v1/flipt.io/Flag/v1alpha1/default/chat-enabled | jq .
{
  "key": "chat-enabled",
  "name": "Chat Enabled",
  "description": "Enable chat for all users",
  "enabled": true,
  "variants": null,
  "rules": null
}
```

#### Listing Flags

```sh
➜  curl --silent localhost:9191/api/v1/flipt.io/Flag/v1alpha1/default | jq .
[
  {
    "key": "chat-enabled",
    "name": "Chat Enabled",
    "description": "Enable chat for all users",
    "enabled": true,
    "variants": null,
    "rules": null
  },
  ...
]
```

#### Putting Flags

> Currently, this only creates commits and pushes the branch, it doesn't open the PR. That is to come.

```sh
➜  curl --silent -H 'Content-Type: application/json' -X PUT localhost:9191/api/v1/flipt.io/Flag/v1alpha1/default --data "{\"key\":\"foo\",\"name\":\"Foo\"}" | jq .
{
  "status": "",
  "id": "179c1c92-cb81-4f30-a1bd-b5c86d8928da"
}
```

#### Deleting Flags

```sh
➜  curl --silent -H 'Content-Type: application/json' -X DELETE localhost:9191/api/v1/flipt.io/Flag/v1alpha1/default/chat-enabled | jq .
{
  "status": "",
  "id": "e7ad4fc9-aa1f-42b4-a129-71c2bc67cfc4"
}
```

## Goals

It will combine Git and WASM to create a ClickOps experience for your GitOps workflows.
Simply use an off-the-shelf collection runtime for your favourite configuration format, or build your own in the language of your choice.

## Collection Runtimes

A collection runtime provides a materialized view over logical collections of resources in your repositories.
`fidgit` can be configured to invoke implementations of its runtime WASM ABI to introspect types, list contents and make changes.
It ships with a user iterface, which leverages the introspection capabilities to present a configurable search and editing experience in your browser.

Any mutations made by the runtime are packaged into Git commits, pushed to a target upstream.
When SCM (GitHub, Gitlab etc.) access is configured, pull or merge requests can be automatically opened based on these proposed changes.
