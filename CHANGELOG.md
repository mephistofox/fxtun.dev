# Changelog

## [1.12.0](https://github.com/mephistofox/fxTunnel/compare/v1.11.1...v1.12.0) (2026-01-30)


### Features

* **tunnel:** add stability and performance improvements ([1e95e79](https://github.com/mephistofox/fxTunnel/commit/1e95e79d0a75e11e59231137ff3bec20fec0eff5))


### Bug Fixes

* remove unused limitedWriter and improve rate limiter cleanup ([c72abe0](https://github.com/mephistofox/fxTunnel/commit/c72abe0a583de71126a3420066b9458007542496))

## [1.11.1](https://github.com/mephistofox/fxTunnel/compare/v1.11.0...v1.11.1) (2026-01-30)


### Bug Fixes

* **gui:** add typed wails bindings for InspectService ([8f1d726](https://github.com/mephistofox/fxTunnel/commit/8f1d726e57baef53ff1cb624e4711caaa5921c86))
* **gui:** correct wails import paths and add InspectService bindings ([acfc949](https://github.com/mephistofox/fxTunnel/commit/acfc949969de6cc6cef5f153b4a4b9a6d1417ee3))
* **web:** use correct localStorage key for SSE auth token ([4fa1a80](https://github.com/mephistofox/fxTunnel/commit/4fa1a80c7a68e1d5895692b44d4685a208c5ba6f))

## [1.11.0](https://github.com/mephistofox/fxTunnel/compare/v1.10.2...v1.11.0) (2026-01-30)


### Features

* **api:** add inspect handlers (list, detail, clear, SSE stream) ([efa4863](https://github.com/mephistofox/fxTunnel/commit/efa4863d2b7149cd3e25c8ba4681813d313f8fbe))
* **api:** add InspectProvider interface and inspect routes ([ce27c99](https://github.com/mephistofox/fxTunnel/commit/ce27c9959c655731326207b6d60ad46144195d83))
* **config:** add inspect settings ([a611bb1](https://github.com/mephistofox/fxTunnel/commit/a611bb1498b2f428e84153844624f133f76cd179))
* **gui:** add traffic inspection view with SSE streaming ([da2a494](https://github.com/mephistofox/fxTunnel/commit/da2a4945a1051464a02ac28c71b6dfb0e497cd15))
* **inspect:** add CapturedExchange data model ([a6bb61b](https://github.com/mephistofox/fxTunnel/commit/a6bb61ba042d1a71d7676cf0ec7d0dd474d2f36d))
* **inspect:** add Manager for per-tunnel buffers ([b577e7e](https://github.com/mephistofox/fxTunnel/commit/b577e7eccc881f06ed833fc44fb0af948833cb3d))
* **inspect:** add RingBuffer with fan-out subscriptions ([5a56185](https://github.com/mephistofox/fxTunnel/commit/5a56185f495b56af15ce86b1d84c4eb08461b0ca))
* **inspect:** capture HTTP traffic in HandleConnection ([5023c3b](https://github.com/mephistofox/fxTunnel/commit/5023c3bc2cb5f86ea8596dcef70edc77b624f1fc))
* **server:** integrate InspectManager into server lifecycle ([45193c8](https://github.com/mephistofox/fxTunnel/commit/45193c84289a1a189f66cf702c7aaadf03535f9e))
* **server:** wire InspectManager into API server ([83ef74f](https://github.com/mephistofox/fxTunnel/commit/83ef74fa110fd8015f24600f0249105be14c33f7))
* **web:** add traffic inspection UI with real-time SSE ([dffe5c2](https://github.com/mephistofox/fxTunnel/commit/dffe5c24085f6aec6b27e2ece6aadf6bc7cd1615))

## [1.10.2](https://github.com/mephistofox/fxTunnel/compare/v1.10.1...v1.10.2) (2026-01-30)


### Performance Improvements

* **server:** add yamux stream pool for low-latency connection handling ([d16564a](https://github.com/mephistofox/fxTunnel/commit/d16564ab9e802498024fd2795e6fbffbdff2d58e))

## [1.10.1](https://github.com/mephistofox/fxTunnel/compare/v1.10.0...v1.10.1) (2026-01-30)


### Bug Fixes

* **lint:** use pointer types in sync.Pool, simplify select to sleep ([06c30c2](https://github.com/mephistofox/fxTunnel/commit/06c30c22903d31b0c7f69622b9d0359ffdb75570))


### Performance Improvements

* **proxy:** optimize tunnel proxying for minimal overhead ([3e87fcb](https://github.com/mephistofox/fxTunnel/commit/3e87fcb4c0368cab0ba4b75d732dc942ab2d5006))

## [1.10.0](https://github.com/mephistofox/fxTunnel/compare/v1.9.1...v1.10.0) (2026-01-30)


### Features

* **cli:** rename --subdomain to --domain, add login/logout commands ([89f7666](https://github.com/mephistofox/fxTunnel/commit/89f766651d10e5d43dd658cb16392b74219a48f4))

## [1.9.1](https://github.com/mephistofox/fxTunnel/compare/v1.9.0...v1.9.1) (2026-01-29)


### Bug Fixes

* **ci:** filter artifact download to exclude docker buildx internals ([fa4ac74](https://github.com/mephistofox/fxTunnel/commit/fa4ac7437e4a7cf7194b7569faecf852121b5d40))

## [1.9.0](https://github.com/mephistofox/fxTunnel/compare/v1.8.3...v1.9.0) (2026-01-29)


### Features

* **server:** add i18n, embed templates, admin bypass for interstitial ([03b49dc](https://github.com/mephistofox/fxTunnel/commit/03b49dc7c0b86b974e8e20549c6be6010a6af4d8))

## [1.8.3](https://github.com/mephistofox/fxTunnel/compare/v1.8.2...v1.8.3) (2026-01-29)


### Bug Fixes

* **client:** wait for goroutines before reconnect to prevent WaitGroup panic ([1d0e177](https://github.com/mephistofox/fxTunnel/commit/1d0e1771fae16382aa9ac84b22bf2f865af998d6))

## [1.8.2](https://github.com/mephistofox/fxTunnel/compare/v1.8.1...v1.8.2) (2026-01-29)


### Bug Fixes

* **docker:** bump Go image to 1.24 to match go.mod requirement ([4239bcb](https://github.com/mephistofox/fxTunnel/commit/4239bcb3022695f4efe362794eec7efd8609cd28))

## [1.8.1](https://github.com/mephistofox/fxTunnel/compare/v1.8.0...v1.8.1) (2026-01-29)


### Bug Fixes

* **ci:** lowercase docker image name for ghcr.io compatibility ([5fc66f9](https://github.com/mephistofox/fxTunnel/commit/5fc66f98621c377ea6f78cd24d6766afdbe95a62))

## [1.8.0](https://github.com/mephistofox/fxTunnel/compare/v1.7.0...v1.8.0) (2026-01-29)


### Features

* **server:** add interstitial warning page for HTTP tunnels ([5f45ec3](https://github.com/mephistofox/fxTunnel/commit/5f45ec379662d6734192a96cec0e5fe9675b83fc))


### Bug Fixes

* **gui:** disable systray on Windows to prevent window closing after auth ([25e06c5](https://github.com/mephistofox/fxTunnel/commit/25e06c5b536dc0fb4e78af5f874d386706b2e6c1))
* **web:** use mfdev.ru/install.sh instead of get.mfdev.ru subdomain ([16bf098](https://github.com/mephistofox/fxTunnel/commit/16bf098187b3d883698b9e6be8e0610aea1fea25))

## [1.7.0](https://github.com/mephistofox/fxTunnel/compare/v1.6.0...v1.7.0) (2026-01-29)


### Features

* add IP whitelist on API tokens for control plane auth ([a5bb9fc](https://github.com/mephistofox/fxTunnel/commit/a5bb9fc0deeb71818b3ee986e4ef61c737d50a23))
* add Prometheus metrics endpoint and request instrumentation ([74511e7](https://github.com/mephistofox/fxTunnel/commit/74511e70246669dc3e157da9e4161d2c5937f218))
* add real-time traffic stats for tunnels ([984c468](https://github.com/mephistofox/fxTunnel/commit/984c4689f5b77ec544d7d6f86bdfeed1d1f2680a))
* **gui:** add auto token refresh, blocked user screen, and support link ([117ce91](https://github.com/mephistofox/fxTunnel/commit/117ce9127627a7d10b42359dc684fdb62ae5c344))
* **gui:** add system tray, log forwarding, build info, and auto-connect ([fa37bf7](https://github.com/mephistofox/fxTunnel/commit/fa37bf7f077e551a3d03cba8596a545e80d65a6e))
* **security:** add per-IP rate limiting for API endpoints ([a0a5876](https://github.com/mephistofox/fxTunnel/commit/a0a58762f8bb5b8d165264e2d844d9152878ea74))
* **security:** add security response headers middleware ([bc6c3ca](https://github.com/mephistofox/fxTunnel/commit/bc6c3ca3b3313d9a4cb4832ccfe8cc8c58bbbe9c))
* **security:** replace wildcard CORS with configurable origins ([490d77f](https://github.com/mephistofox/fxTunnel/commit/490d77fc856a417fb6d269676d7dcc220fc6ef6b))
* **security:** require jwt_secret and totp_key when web panel is enabled ([a7ca5e2](https://github.com/mephistofox/fxTunnel/commit/a7ca5e2a08720b1a28a8e4163ac7e3f3ec7dadb0))
* **web:** add dynamic version display and i18n additions ([4e87e1c](https://github.com/mephistofox/fxTunnel/commit/4e87e1c27e75a575eac245bb867eece72887c885))
* **web:** add install script, download section with platform picker, and update domains ([a0f220a](https://github.com/mephistofox/fxTunnel/commit/a0f220a73e12e6ca9c24812953812cbadcb11fce))


### Bug Fixes

* **build:** pass ldflags to wails dev and build commands ([5a00d18](https://github.com/mephistofox/fxTunnel/commit/5a00d182d0b1b8736fda58fe64beb28fdaa342ae))
* **ci:** add BuildTime to GUI ldflags and update Go to 1.24 ([9e963eb](https://github.com/mephistofox/fxTunnel/commit/9e963ebb56d35441e24a4c05f51411383ad11e84))
* **gui:** improve layout compactness and fix UI issues ([33fc4de](https://github.com/mephistofox/fxTunnel/commit/33fc4de13efe2bcd34ddd2e1d8f2282f9d04c4cf))
* **gui:** use format string in fmt.Errorf calls ([d2c0ad4](https://github.com/mephistofox/fxTunnel/commit/d2c0ad4788f77a5fc09231326ce82da16e826bf2))
* resolve critical race conditions, goroutine leaks, and performance issues ([2832875](https://github.com/mephistofox/fxTunnel/commit/2832875f17a9436897623682c47cf763c0d2847e))
* resolve golangci-lint errors and tidy go.mod ([af266ba](https://github.com/mephistofox/fxTunnel/commit/af266ba3698f226c0e8d776832b4a6e9d6158608))
* **web:** fix mobile overflow issues on landing page ([2050f9c](https://github.com/mephistofox/fxTunnel/commit/2050f9cb37d96931b41ef697964e8e6081628b55))
* **web:** reduce section spacing on mobile and hide Learn More button ([0dd32c3](https://github.com/mephistofox/fxTunnel/commit/0dd32c3bea73676e0522ca0e1bd119e93a78a4e8))
* **web:** show nav background when mobile menu is open at top ([2ec6bf5](https://github.com/mephistofox/fxTunnel/commit/2ec6bf56557852ca5eae28463b2bebc968e2eefa))


### Performance Improvements

* **client:** cache resolved local address to skip IPv4/IPv6 probe ([f282bf7](https://github.com/mephistofox/fxTunnel/commit/f282bf77d940f59f5040e7d0e116dc74c8505583))
* **client:** race IPv4/IPv6 in parallel and pre-probe on tunnel creation ([875df70](https://github.com/mephistofox/fxTunnel/commit/875df7025151b8ebae4b83fbf6f440e33c2ec490))

## [1.6.0](https://github.com/mephistofox/fxTunnel/compare/v1.5.0...v1.6.0) (2026-01-28)


### Features

* **gui:** complete redesign with cyber-industrial theme ([39bb251](https://github.com/mephistofox/fxTunnel/commit/39bb251628e3f2b40d963dced4f5810d752a8256))

## [1.5.0](https://github.com/mephistofox/fxTunnel/compare/v1.4.0...v1.5.0) (2026-01-28)


### Features

* **gui:** redesign to cyber-industrial theme ([758fe35](https://github.com/mephistofox/fxTunnel/commit/758fe35f4bea1b7feb63431f984bc399b9c7377a))


### Bug Fixes

* **server:** prevent 404 page layout shift on mobile ([4109f81](https://github.com/mephistofox/fxTunnel/commit/4109f81b3bfc06dc4967b0f0b980559888a12029))

## [1.4.0](https://github.com/mephistofox/fxTunnel/compare/v1.3.1...v1.4.0) (2026-01-27)


### Features

* **client:** add automatic JWT token refresh on reconnect ([1feff99](https://github.com/mephistofox/fxTunnel/commit/1feff9990a6bd0c937e0241c30e53f72acb752fd))
* **server:** add custom styled 404 error page ([2b5d08a](https://github.com/mephistofox/fxTunnel/commit/2b5d08ab24d122fc507b22d9d2b520dd87a9a177))
* **web:** redesign landing page with modern cyber-industrial theme ([2f76e15](https://github.com/mephistofox/fxTunnel/commit/2f76e15a8791382c168985386d56e3e428ee751c))
* **web:** update landing page fonts and add animated card borders ([fdcb6d7](https://github.com/mephistofox/fxTunnel/commit/fdcb6d7d5712ddb524a30af5c6baf8d410c2633e))

## [1.3.1](https://github.com/mephistofox/fxTunnel/compare/v1.3.0...v1.3.1) (2026-01-26)


### Bug Fixes

* **build:** clean old files before building to downloads/ ([667e278](https://github.com/mephistofox/fxTunnel/commit/667e278f9dc8201d4fffd4fd2dfa1c89b0855788))

## [1.3.0](https://github.com/mephistofox/fxTunnel/compare/v1.2.0...v1.3.0) (2026-01-26)


### Features

* **gui:** add refresh token support for persistent sessions ([c3b6eba](https://github.com/mephistofox/fxTunnel/commit/c3b6eba8ff894e4c2cc9d96f24f9750520b91c39))


### Bug Fixes

* **ci:** make downloads sync non-fatal when assets not ready yet ([f0da8f4](https://github.com/mephistofox/fxTunnel/commit/f0da8f492d666eff63f86f43f667ad800b3fc520))
* **gui:** use token method for auto-login to fix session persistence ([04e64fa](https://github.com/mephistofox/fxTunnel/commit/04e64fabcd4df1ffcb239b5498cdda8d0f2f327f))

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
