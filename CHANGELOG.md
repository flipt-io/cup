# Changelog

## 0.1.0 (2023-08-28)


### Features

* add an initial design doc for alpha ([fb03633](https://github.com/flipt-io/cup/commit/fb03633dcd048235376e5a752e8589ad32c872d8))
* add billfs billy to io/fs adaptor ([422baa3](https://github.com/flipt-io/cup/commit/422baa3d234875a6a94dc90d2f2a212535b0bc8f))
* add inmem fs and simple builtin controller ([914366c](https://github.com/flipt-io/cup/commit/914366c6e90bd3082760701fe0fc20d164ca154d))
* add LICENSE ([3d121f8](https://github.com/flipt-io/cup/commit/3d121f836b0f30eee13695ba68853c6748404ab6))
* **api:** add initial server implementation (source, defs, get and list) ([d80a6fb](https://github.com/flipt-io/cup/commit/d80a6fb97ca66e76b623f1893460f07507d4862d))
* **api:** add support for validating JSON schema on PUT ([2237a48](https://github.com/flipt-io/cup/commit/2237a48e1bac4ad1cce49854cdf0f8018ec95542))
* **api:** implement put and delete ([525b0f4](https://github.com/flipt-io/cup/commit/525b0f4f20be677d92aebf9458b98ee3b7c4445f))
* **build:** add ./build.sh hack:fliptcup target ([f46cb54](https://github.com/flipt-io/cup/commit/f46cb540dd263d67138cf0b9e7bbe7cbd99403d4))
* **build:** add ./build.sh publish target ([1dd6a51](https://github.com/flipt-io/cup/commit/1dd6a51e109f6ea494bf479adc9b7e2a76e4b900))
* **build:** publish fliptcup build to GHCR ([0e0aa89](https://github.com/flipt-io/cup/commit/0e0aa899cd7afea3019ebacdd3e8fd0862b13c0b))
* **cmd/cup:** add delete subcommand ([0d75510](https://github.com/flipt-io/cup/commit/0d7551091bc747edadd07f742d3b22af26f9d2a5))
* **cmd/cup:** adds the new cup CLI ([25b314e](https://github.com/flipt-io/cup/commit/25b314eee73c752e67fbe15349590a04c814932b))
* **cmd/cup:** apply command and various fixes ([8ff2d36](https://github.com/flipt-io/cup/commit/8ff2d36ae0974c5bd1e9ef472cad39a1767058e4))
* **cmd/cupd:** support env vars ([fc1caf4](https://github.com/flipt-io/cup/commit/fc1caf4cf27c215d008b4aa3a8b883e1399d486f))
* **cmd/cup:** organize commands into categories ([32c8b7c](https://github.com/flipt-io/cup/commit/32c8b7c4a600caefff97f25f12e488f5f6258e44))
* **cmd/cup:** print PR on edit and apply ([a1a607d](https://github.com/flipt-io/cup/commit/a1a607df3b273c21ab721a9e32800e6e2cf25990))
* **cmd/cup:** support json output for cup config context ([c4983ef](https://github.com/flipt-io/cup/commit/c4983efbd20e3bed561ae789c0148c8de11ad1df))
* **containers:** add convenience MapStore ([3b65d6b](https://github.com/flipt-io/cup/commit/3b65d6bffe652af2ec25a83598515ddad5344170))
* **controllers/flipt.io/v1alpha:** add controller ([57b40d4](https://github.com/flipt-io/cup/commit/57b40d4cfeb0ffed805a8ddce7410f5c6f991e38))
* **controllers/flipt.io/v1alpha:** add Flag kind controller ([57b40d4](https://github.com/flipt-io/cup/commit/57b40d4cfeb0ffed805a8ddce7410f5c6f991e38))
* **controllers/flipt.io/v1alpha:** add Segment kind controller ([57b40d4](https://github.com/flipt-io/cup/commit/57b40d4cfeb0ffed805a8ddce7410f5c6f991e38))
* **cup/ctl:** add identation to json output ([57b40d4](https://github.com/flipt-io/cup/commit/57b40d4cfeb0ffed805a8ddce7410f5c6f991e38))
* **cup:** support cup edit ([cc9cb2a](https://github.com/flipt-io/cup/commit/cc9cb2af54a61ea4a278443ab69405e59f99be36))
* **docs/site:** add nextjs docs site ([2d17491](https://github.com/flipt-io/cup/commit/2d174918184a18481a3ac28ae92b942169a30310))
* **docs:** add PUT sequence diagram ([ad2a8e1](https://github.com/flipt-io/cup/commit/ad2a8e10678a4fab5ae1f14ef432013d05a09cda))
* **docs:** add resource definition controller and path ([a3d6e77](https://github.com/flipt-io/cup/commit/a3d6e779a3fdbe3ea5a72f4e85048405a22ae322))
* **flipt:** implement collection delete ([ae5f065](https://github.com/flipt-io/cup/commit/ae5f065606a3379039bed860583f928c1153c308))
* **git/scm:** implement github type SCM ([4932708](https://github.com/flipt-io/cup/commit/493270867591abc0eb8004c0f150fe019c7fca5f))
* **gitea:** add gitea SCM source wrapper for PR creation ([3223c37](https://github.com/flipt-io/cup/commit/3223c371a261e5085138c9d18b60fb8400fe71eb))
* **github:** push both cup/cupd and cup/flipt ([cf92792](https://github.com/flipt-io/cup/commit/cf92792ebe5e9351951c5cbd8ddba7b1ddbbba93))
* implement cupd serve ([e35d7bb](https://github.com/flipt-io/cup/commit/e35d7bb05733861fcd01f6c1fb1eb81c713e492c))
* implement wasm runtime executor ([5643ce6](https://github.com/flipt-io/cup/commit/5643ce653904be34c37a6c2f5386f301d758240a))
* initial constructs for fidgit ([7c3ea8e](https://github.com/flipt-io/cup/commit/7c3ea8ec5adf5bcc6740a16974f32ec94f902cb3))
* initial implementation of git.FilesystemStore ([984c94c](https://github.com/flipt-io/cup/commit/984c94c17924cf8ce515ea018803c161f8b75f28))
* **pkg/controller:** add fs config structure ([c73e021](https://github.com/flipt-io/cup/commit/c73e021fc161f4241eee23765075367df97069d8))
* **pkg/fs/git:** add SCM abstraction and tests ([3dcea58](https://github.com/flipt-io/cup/commit/3dcea58313aec5b5b68405660cf8c9310f4de90c))
* support local source mutations ([36826e0](https://github.com/flipt-io/cup/commit/36826e0018557ea7d831b79e3d68dc3b361678e5))
* support mutations to git source ([33270b6](https://github.com/flipt-io/cup/commit/33270b6c1f0be25b9b27903949f7840c8dcdae17))
* update README ([f5e37fc](https://github.com/flipt-io/cup/commit/f5e37fc5dcfe68ad5fcf5163836f8e656542309d))
* use billy.Filesystem and test server get ([c2d3b2c](https://github.com/flipt-io/cup/commit/c2d3b2cb67dd549763b96f2743e193c149313e34))
* **wasm:** implement WASM controller GET ([56a6d6a](https://github.com/flipt-io/cup/commit/56a6d6a8dc6e2dea70afba94f1b0905dd6862744))


### Bug Fixes

* **api:** define a valid schema in unit tests ([0dc8f6f](https://github.com/flipt-io/cup/commit/0dc8f6f21a156e0bf22da61052602ba7e3a8f37b))
* **build:** add cup to path in base for integration tests ([a677136](https://github.com/flipt-io/cup/commit/a6771364229448198ba24da9d7b81751c75cbd40))
* **build:** correct flag type json schema ([ddc98f5](https://github.com/flipt-io/cup/commit/ddc98f50c59f8a05bb748401568d793ca7b3c930))
* **build:** image export only for clients default platform ([9621e4f](https://github.com/flipt-io/cup/commit/9621e4f51b9da1c5483f33b5ac725a54c3ac5f06))
* **cmd/cup:** add usage ([8b984c7](https://github.com/flipt-io/cup/commit/8b984c76a5b48242b6f44d3758b73821c44f16cb))
* **cmd/cupd:** set options on subcommands ([31e5141](https://github.com/flipt-io/cup/commit/31e5141a2ee66a6627fe628513dd9d58d79fbac9))
* dont push and propose on empty commit ([e01ffd4](https://github.com/flipt-io/cup/commit/e01ffd41d60a32a6fbc9ee47ac3a4c79044fdaf1))
* **ext/controller/flipt:** correct schema for recent validation changes ([f7d8d31](https://github.com/flipt-io/cup/commit/f7d8d31061581b7bc9a9e99b35debd0d03961470))
* **gh/test:** install dagger CLI ([a2c4dac](https://github.com/flipt-io/cup/commit/a2c4dac0112d68114007e8096cca8bbc8a17aeb6))
* **gitfs:** correct signature in unit test ([8c42e7a](https://github.com/flipt-io/cup/commit/8c42e7a1c28ee620a524be013f5e5a3b194e9dd5))
* **github:** use cross compilation and fix bad image name ([3a530d1](https://github.com/flipt-io/cup/commit/3a530d18bb0dcddbae11ea838c026f985d46b162))
* **git:** use latest wazero and update gitfs to impl Readdir ([2bfc707](https://github.com/flipt-io/cup/commit/2bfc707e4061837e982da37550db5a2abbcfffad))
* **source/git:** temporarily remove gitfs in favour of tmp dir ([56404c9](https://github.com/flipt-io/cup/commit/56404c9cbe871874fcfc3475ac756d561e137357))
* **source/git:** use fork of wazero with fs.FS.ReadDir fix ([9efba11](https://github.com/flipt-io/cup/commit/9efba11e2ce6353ace23c93615238806d8cfb0b8))


### Miscellaneous Chores

* set release version to 0.1.0 ([5532a3c](https://github.com/flipt-io/cup/commit/5532a3c7175998bf646d42cf898aa9f81919002d))
