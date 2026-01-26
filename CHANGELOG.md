# Changelog

## [1.2.0](https://github.com/mephistofox/fxTunnel/compare/v1.1.1...v1.2.0) (2026-01-26)


### Features

* **ci:** sync client downloads from latest release on deploy ([ed3ef19](https://github.com/mephistofox/fxTunnel/commit/ed3ef19ac789ca7a79467338e9bc7f8f66f65cdf))


### Bug Fixes

* **ci:** remove conflicting --skip-existing flag from gh release download ([6de8a31](https://github.com/mephistofox/fxTunnel/commit/6de8a31221c3814a8ed7c19a09eed79d54225615))

## [1.1.1](https://github.com/mephistofox/fxTunnel/compare/v1.1.0...v1.1.1) (2026-01-26)


### Bug Fixes

* **ci:** enable CGO for server build (required by go-sqlite3) ([88fce88](https://github.com/mephistofox/fxTunnel/commit/88fce887dcc2f3327173f8553098861d2ffc5284))
* **ci:** use workflow_run trigger for deploy instead of wait-on-check ([62dd81a](https://github.com/mephistofox/fxTunnel/commit/62dd81a436a2e7fecaab8b88c0784fdd6b96a497))
* **client:** add IPv4/IPv6 fallback for local service connections ([766ba87](https://github.com/mephistofox/fxTunnel/commit/766ba871e1fb513bc2f1b2b5a7fc02a86877d4a3))

## [1.1.0](https://github.com/mephistofox/fxTunnel/compare/v1.0.0...v1.1.0) (2025-12-25)


### Features

* **admin:** add admin panel with tunnels, audit logs and user management ([2c4bb1c](https://github.com/mephistofox/fxTunnel/commit/2c4bb1cf3050ed6775081a926d5506ef1170f444))
* **gui,web:** add management panel and desktop application ([536e074](https://github.com/mephistofox/fxTunnel/commit/536e07426333dedcd3eea4b2fccb6b5c71b88a22))
* initial implementation of fxTunnel reverse tunneling system ([9e851ab](https://github.com/mephistofox/fxTunnel/commit/9e851ab38d60d94ed9b15e80f796b7f61a909bfd))
* **sync:** add data synchronization between GUI and server ([0812d4c](https://github.com/mephistofox/fxTunnel/commit/0812d4cc3b16825cef9fba786749a35fcb7a141f))


### Bug Fixes

* **ci:** build frontends before tests and lint ([fc81616](https://github.com/mephistofox/fxTunnel/commit/fc81616b65c7cefb9609674be70a500771afcd1f))
* **ci:** disable errcheck linter, use exclude-dirs ([8b30028](https://github.com/mephistofox/fxTunnel/commit/8b300288d08a56c09ff509c75126171d8927545b))
* **ci:** exclude GUI from CI checks, use placeholder dist ([4a3d7aa](https://github.com/mephistofox/fxTunnel/commit/4a3d7aad6735c653d171bd1200cbfeb9cd6a6b01))
* **ci:** explicitly disable errcheck linter ([c3359c7](https://github.com/mephistofox/fxTunnel/commit/c3359c7e4d869da864986ca8f86b9244bd9af190))
* **ci:** handle different Wails output paths on macOS ([24ecb86](https://github.com/mephistofox/fxTunnel/commit/24ecb86b709742186b3d37938b5070e7465b8008))
* remove unused variables and apply gosimple suggestion ([03da85c](https://github.com/mephistofox/fxTunnel/commit/03da85c25a78b18a24191cd887b2f1510f9b878d))

## 1.0.0 (2025-12-25)


### Features

* **admin:** add admin panel with tunnels, audit logs and user management ([2c4bb1c](https://github.com/mephistofox/fxTunnel/commit/2c4bb1cf3050ed6775081a926d5506ef1170f444))
* **gui,web:** add management panel and desktop application ([536e074](https://github.com/mephistofox/fxTunnel/commit/536e07426333dedcd3eea4b2fccb6b5c71b88a22))
* initial implementation of fxTunnel reverse tunneling system ([9e851ab](https://github.com/mephistofox/fxTunnel/commit/9e851ab38d60d94ed9b15e80f796b7f61a909bfd))
* **sync:** add data synchronization between GUI and server ([0812d4c](https://github.com/mephistofox/fxTunnel/commit/0812d4cc3b16825cef9fba786749a35fcb7a141f))


### Bug Fixes

* **ci:** build frontends before tests and lint ([fc81616](https://github.com/mephistofox/fxTunnel/commit/fc81616b65c7cefb9609674be70a500771afcd1f))
* **ci:** disable errcheck linter, use exclude-dirs ([8b30028](https://github.com/mephistofox/fxTunnel/commit/8b300288d08a56c09ff509c75126171d8927545b))
* **ci:** exclude GUI from CI checks, use placeholder dist ([4a3d7aa](https://github.com/mephistofox/fxTunnel/commit/4a3d7aad6735c653d171bd1200cbfeb9cd6a6b01))
* **ci:** explicitly disable errcheck linter ([c3359c7](https://github.com/mephistofox/fxTunnel/commit/c3359c7e4d869da864986ca8f86b9244bd9af190))
* **ci:** handle different Wails output paths on macOS ([24ecb86](https://github.com/mephistofox/fxTunnel/commit/24ecb86b709742186b3d37938b5070e7465b8008))
* remove unused variables and apply gosimple suggestion ([03da85c](https://github.com/mephistofox/fxTunnel/commit/03da85c25a78b18a24191cd887b2f1510f9b878d))

## 1.0.0 (2025-12-25)


### Features

* **admin:** add admin panel with tunnels, audit logs and user management ([8f4ac55](https://github.com/mephistofox/fxTunnel/commit/8f4ac55d5de0b533515c74814e248259fc26bd96))
* **gui,web:** add management panel and desktop application ([536e074](https://github.com/mephistofox/fxTunnel/commit/536e07426333dedcd3eea4b2fccb6b5c71b88a22))
* initial implementation of fxTunnel reverse tunneling system ([9e851ab](https://github.com/mephistofox/fxTunnel/commit/9e851ab38d60d94ed9b15e80f796b7f61a909bfd))
* **sync:** add data synchronization between GUI and server ([af415a4](https://github.com/mephistofox/fxTunnel/commit/af415a48959b40574a8d801443053d05a07682c1))


### Bug Fixes

* **ci:** build frontends before tests and lint ([7cf0ff9](https://github.com/mephistofox/fxTunnel/commit/7cf0ff9e15c563c15f3476b719db429447d08fd0))
* **ci:** disable errcheck linter, use exclude-dirs ([e0d1af8](https://github.com/mephistofox/fxTunnel/commit/e0d1af8677d8a0e136b265eeaa9016e9adf58bf8))
* **ci:** exclude GUI from CI checks, use placeholder dist ([4927b8c](https://github.com/mephistofox/fxTunnel/commit/4927b8c3e34b0ce73190f81e211a7831901f7996))
* **ci:** explicitly disable errcheck linter ([4485f22](https://github.com/mephistofox/fxTunnel/commit/4485f223e4449267d604eb1a62aaae97d2a8d075))
* remove unused variables and apply gosimple suggestion ([b652e8b](https://github.com/mephistofox/fxTunnel/commit/b652e8b03a858f0b54eb21c9612393e4f695be09))
