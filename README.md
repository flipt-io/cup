fidgit - The GitOps API Framework
---------------------------------

`fidgit` brings configuration repositories to life.

At its core it is a specification for cataloging and transforming the contents of Git repositories.
It combines Git and WASM to create a ClickOps experience for your GitOps workflows.
Simply use an off-the-shelf collection runtime for your favourite configuration format, or build your own in the language of your choice.

## Collection Runtimes

A collection runtime provides a materialized view over logical collections of resources in your repositories.
`fidgit` can be configured to invoke implementations of its runtime WASM ABI to introspect types, list contents and make changes.
It ships with a user iterface, which leverages the introspection capabilities to present a configurable search and editing experience in your browser.

Any mutations made by the runtime are packaged into Git commits, pushed to a target upstream.
When SCM (GitHub, Gitlab etc.) access is configured, pull or merge requests can be automatically opened based on these proposed changes.
