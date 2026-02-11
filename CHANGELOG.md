# Changelog

## [3.2.0](https://github.com/mephistofox/fxtun.dev/compare/v3.1.0...v3.2.0) (2026-02-08)


### Features

* **web:** add OG meta tags, canonical/hreflang, 404 page, WebSite schema and font preloads ([5094661](https://github.com/mephistofox/fxtun.dev/commit/5094661d9c41dade401d3b793c76153e38d6a74e))
* **web:** replace grid overlay with animated topographic contours ([89c3e14](https://github.com/mephistofox/fxtun.dev/commit/89c3e145e9ff98db72dcbf54184b64a89d3464b9))


### Bug Fixes

* **web:** add light theme support for code blocks and fix SSG theme flash ([d225c7a](https://github.com/mephistofox/fxtun.dev/commit/d225c7aedc7de52a611325aaf4175dc5c3db3d7c))

## [3.1.0](https://github.com/mephistofox/fxtun.dev/compare/v3.0.0...v3.1.0) (2026-02-08)


### Features

* **seo:** add favicon suite, OG image, and webmanifest ([d80f35f](https://github.com/mephistofox/fxtun.dev/commit/d80f35f88c87b7911b12d5ea355ecd34ffac4187))
* **web:** integrate vite-ssg for static site generation ([9e7d52a](https://github.com/mephistofox/fxtun.dev/commit/9e7d52ad6aa52a106f4a5024b7f48c4746b235d5))


### Bug Fixes

* **api:** derive install script domain from request Host header ([2064410](https://github.com/mephistofox/fxtun.dev/commit/20644109eeb088b0e5c239f5d1edc44a395ce21e))

## [3.0.0](https://github.com/mephistofox/fxtun.dev/compare/v2.12.0...v3.0.0) (2026-02-08)


### ⚠ BREAKING CHANGES

* **payment:** Robokassa integration removed, YooKassa config required

### Features

* add plan usage display and simplify GUI auth flow ([24ed6ce](https://github.com/mephistofox/fxtun.dev/commit/24ed6ce673f60d00b0e79264c80e6fa4a5c37882))
* **inspect:** add persistent exchange storage and enhanced replay ([d1c34ea](https://github.com/mephistofox/fxtun.dev/commit/d1c34eac111ceed70b2f8ec98844923642963625))
* **payment:** migrate from Robokassa to YooKassa ([5b9c66d](https://github.com/mephistofox/fxtun.dev/commit/5b9c66d97f4bc23effe6425887a1fcc3e2a08b0b))
* remove invite code system ([b979943](https://github.com/mephistofox/fxtun.dev/commit/b97994397dcf02bf8a45e3ff97cf88147a5fe85d))
* **seo:** add @unhead/vue with dynamic meta tags and OG/Twitter cards ([acd5de4](https://github.com/mephistofox/fxtun.dev/commit/acd5de43282841f18a59b23f2030cb1c9a0f3484))
* **seo:** add FAQ section with 15 questions and FAQPage JSON-LD ([9cfaacf](https://github.com/mephistofox/fxtun.dev/commit/9cfaacf90ebb2263a52a59b44dd731e7d61b88e2))
* **seo:** add JSON-LD structured data (Organization, SoftwareApplication) ([42abbaf](https://github.com/mephistofox/fxtun.dev/commit/42abbaf2438e132a0f0eb1d90f3c8fec4f545951))
* **seo:** add robots.txt and llms.txt ([8327760](https://github.com/mephistofox/fxtun.dev/commit/83277604c89501caa3133853452c3c7713fa560a))
* **seo:** add sitemap.xml generation via vite-plugin-sitemap ([1075527](https://github.com/mephistofox/fxtun.dev/commit/107552765799989a9258e2a7f9ce1c9a27802c8b))
* **server:** support domain aliases for subdomain extraction ([feb9e9c](https://github.com/mephistofox/fxtun.dev/commit/feb9e9c500d1d581808d6376495ee3f3b543cb36))
* **web:** add logout button to mobile navigation ([a636f7e](https://github.com/mephistofox/fxtun.dev/commit/a636f7e1c55320d1f35b5a390d7ce8c758f18dd9))
* **web:** optimize landing page structure for conversion ([fc2856d](https://github.com/mephistofox/fxtun.dev/commit/fc2856db661134f247666f65927ee6d8d350d784))
* **web:** redesign inspector and domain setup landing demos ([da1ef9d](https://github.com/mephistofox/fxtun.dev/commit/da1ef9de3d7085205ea92fd54a01d19184caf34a))
* **web:** redesign landing page for SaaS positioning ([0cd5498](https://github.com/mephistofox/fxtun.dev/commit/0cd5498f7a017557a3d8eed5f746b26428531661))


### Bug Fixes

* **email:** remove self-hosted branding from email templates ([6d918f9](https://github.com/mephistofox/fxtun.dev/commit/6d918f974d2d1c3ab69c2b09f726acf63de3b2b7))
* **gui:** update wails bindings and fix build errors ([902a94f](https://github.com/mephistofox/fxtun.dev/commit/902a94f1385787a5848e967e0c2acbebd15f688f))
* **inspect:** fix SSE auth security and upgrade request detection ([91d9b37](https://github.com/mephistofox/fxtun.dev/commit/91d9b37da2e97a3d1e12deeb356f74260173f3c7))
* **inspect:** make exchange persistence synchronous ([90b8654](https://github.com/mephistofox/fxtun.dev/commit/90b8654a40b96bc30d4f34335e5906042f67438c))
* **inspect:** stable cross-restart data with UTF-8 body decoding ([015a322](https://github.com/mephistofox/fxtun.dev/commit/015a3225934f2fade0a707523fa07a3ccbcdab67))
* **inspect:** use crypto/rand for exchange IDs to prevent collisions ([934d1ee](https://github.com/mephistofox/fxtun.dev/commit/934d1ee8a229fcd022e8d1a2f14a69edbf5b051e))
* **payment:** correct amount parsing in success email notification ([d630bfb](https://github.com/mephistofox/fxtun.dev/commit/d630bfbffea5c16e10915751a8e0f71628603c01))
* **server:** simplify isUpgradeRequest to check Connection header only ([4d6f8b0](https://github.com/mephistofox/fxtun.dev/commit/4d6f8b0b529956cafe18f7c2eb4fad5bf508a97b))


### Performance Improvements

* **web:** self-host fonts, remove Google Fonts CDN ([22e1cfd](https://github.com/mephistofox/fxtun.dev/commit/22e1cfdca7f79d7eaf2675192ed7e5c7411c0162))

## [2.12.0](https://github.com/mephistofox/fxtun.dev/compare/v2.11.0...v2.12.0) (2026-02-05)


### Features

* **email:** redesign email templates to match project style ([4d7920e](https://github.com/mephistofox/fxtun.dev/commit/4d7920e6edc438564e0252a661094c82b9b0babc))


### Bug Fixes

* **admin:** change user plan to Free when canceling subscription ([15c354f](https://github.com/mephistofox/fxtun.dev/commit/15c354f337406afb462b18e38aa7e417d5836da2))
* **email:** use LOGIN auth instead of PLAIN for SMTP ([7ea97ec](https://github.com/mephistofox/fxtun.dev/commit/7ea97ec79384b40dcc486c0c4a687df345574dca))

## [2.11.0](https://github.com/mephistofox/fxtun.dev/compare/v2.10.0...v2.11.0) (2026-02-04)


### Features

* **payment:** send email notification after successful payment ([86855c6](https://github.com/mephistofox/fxtun.dev/commit/86855c62f77dbb205426415f2f2b861a1aaf6ec8))

## [2.10.0](https://github.com/mephistofox/fxtun.dev/compare/v2.9.1...v2.10.0) (2026-02-04)


### Features

* improve subscriptions and payments management ([b5a1698](https://github.com/mephistofox/fxtun.dev/commit/b5a16989f39f1be41113379da443e6c6692272ab))
* **payment:** reject test payments in production mode ([0c45533](https://github.com/mephistofox/fxtun.dev/commit/0c45533158ad69a7f2b33261d07cf8ca074999a5))
* **web:** add subscriptions to admin navigation menu ([7b48221](https://github.com/mephistofox/fxtun.dev/commit/7b48221bc94cd8d8973d11956608be22b4c2d814))
* **web:** mobile admin sidebar and subscription cancel confirm ([b4d532a](https://github.com/mephistofox/fxtun.dev/commit/b4d532ab17db9198e190199afb75293a65c74024))


### Bug Fixes

* **payment:** remove unused getPassword2 function ([59cc154](https://github.com/mephistofox/fxtun.dev/commit/59cc15461af6a6188fdb2cc9994e17e9b6c67192))
* **payment:** use raw OutSum and IsTest flag from Robokassa callback ([9ad59f6](https://github.com/mephistofox/fxtun.dev/commit/9ad59f66e8430f4ec15c48a91d7be54c338efbda))
* **web:** show all admin menu items on mobile ([c8f162f](https://github.com/mephistofox/fxtun.dev/commit/c8f162f720b71acfab446e1e3727f38d89ee4c83))

## [2.9.1](https://github.com/mephistofox/fxtun.dev/compare/v2.9.0...v2.9.1) (2026-02-04)


### Bug Fixes

* **payment:** store RUB amount in payment record to match Robokassa callback ([8e73237](https://github.com/mephistofox/fxtun.dev/commit/8e73237d20e131d4909eaf004074aecd386277ef))

## [2.9.0](https://github.com/mephistofox/fxtun.dev/compare/v2.8.0...v2.9.0) (2026-02-04)


### Features

* **web:** redirect to checkout after login when selecting a plan ([e6ad693](https://github.com/mephistofox/fxtun.dev/commit/e6ad69337d8266491d8781b7ca1a7631c8ad3514))


### Bug Fixes

* **web:** add /payments/* routes for Robokassa redirects ([7656ffc](https://github.com/mephistofox/fxtun.dev/commit/7656ffc9446a0261c431427e5e36abeff9cdc5bc))

## [2.8.0](https://github.com/mephistofox/fxtun.dev/compare/v2.7.0...v2.8.0) (2026-02-04)


### Features

* **payment:** centralize price calculation on backend ([84f4a3e](https://github.com/mephistofox/fxtun.dev/commit/84f4a3e302b2db701f06ac5103d1afd17c459bb4))

## [2.7.0](https://github.com/mephistofox/fxtun.dev/compare/v2.6.0...v2.7.0) (2026-02-04)


### Features

* **payment:** add dynamic USD to RUB exchange rate ([cec0050](https://github.com/mephistofox/fxtun.dev/commit/cec0050560bfc743e7ebb0bfecdb478e415e8345))

## [2.6.0](https://github.com/mephistofox/fxtun.dev/compare/v2.5.0...v2.6.0) (2026-02-04)


### Features

* **admin:** add subscription and payment management ([067abeb](https://github.com/mephistofox/fxtun.dev/commit/067abeb429c365b1f65f7cf473ccaf40fea99b7c))
* **api:** add payment and subscription endpoints ([aad3d0c](https://github.com/mephistofox/fxtun.dev/commit/aad3d0c6340c83ffbd2bb6486f02b756ab78de7c))
* **db:** add subscription and payment models ([fb27fab](https://github.com/mephistofox/fxtun.dev/commit/fb27fab9493b3b7aefd7e8ab577eae16a3724e59))
* **email:** add email notification service ([f3b46cc](https://github.com/mephistofox/fxtun.dev/commit/f3b46ccfcff692ed7b9b8e2ebb302866dc15d903))
* **payment:** add Robokassa integration module ([8944821](https://github.com/mephistofox/fxtun.dev/commit/89448212ffd1eec61cf1f171de5c829234c91438))
* **scheduler:** add subscription lifecycle scheduler ([848aad2](https://github.com/mephistofox/fxtun.dev/commit/848aad2742ab5cb35aa366ebe837c3a4d11f636a))
* **web:** add checkout and payment result pages ([3a3b9e8](https://github.com/mephistofox/fxtun.dev/commit/3a3b9e8cb3f6e44fec8be01fc20b9193d28cb324))
* **web:** add public offer page for Robokassa compliance ([0585885](https://github.com/mephistofox/fxtun.dev/commit/05858858feedb32768a4c84933d5f2093e10c5d2))
* **web:** add subscription section to profile page ([e06a1cb](https://github.com/mephistofox/fxtun.dev/commit/e06a1cb12729813fd5b2838601cd83670f59d825))


### Bug Fixes

* handle errcheck warnings in payment and scheduler ([3baa72c](https://github.com/mephistofox/fxtun.dev/commit/3baa72c12e227c3c4a6fb5ca956885a9746347f5))
* resolve golangci-lint warnings ([f90e639](https://github.com/mephistofox/fxtun.dev/commit/f90e639fffc083ff8bc06a9247700310dca9ea4e))
* **web:** show offer link only on fxtun.ru domain ([6b2e2d0](https://github.com/mephistofox/fxtun.dev/commit/6b2e2d0b4a18725476ea2404ea5c307d28b93a48))

## [2.5.0](https://github.com/mephistofox/fxtun.dev/compare/v2.4.1...v2.5.0) (2026-02-04)


### Features

* **web:** add animated tunnel visualization with color-shifting dots ([7a0da8c](https://github.com/mephistofox/fxtun.dev/commit/7a0da8ca5e1fdc9d7d7af1824d70048da9508486))
* **web:** improve landing and auth UX ([fba8848](https://github.com/mephistofox/fxtun.dev/commit/fba8848e463588c3f6a48d55655aa76620a6728d))

## [2.4.1](https://github.com/mephistofox/fxtun.dev/compare/v2.4.0...v2.4.1) (2026-02-04)


### Bug Fixes

* **web:** show dash icon when domains limit is 0 ([8e2a0f5](https://github.com/mephistofox/fxtun.dev/commit/8e2a0f5f27ccdfa71060fcc35d4ed9c0e55e2726))

## [2.4.0](https://github.com/mephistofox/fxtun.dev/compare/v2.3.0...v2.4.0) (2026-02-04)


### Features

* **web:** add descriptive hints to pricing plan features ([f9b9d1a](https://github.com/mephistofox/fxtun.dev/commit/f9b9d1a616cc765e84beff5cd05cf4512f8f7a47))


### Bug Fixes

* **ci:** use systemctl kill for quick service restart ([dbd7004](https://github.com/mephistofox/fxtun.dev/commit/dbd7004202d833dc2a34e58ddff8bcd88f5edf5f))
* **web:** remove GitHub button from hero, keep Learn More ([7a76868](https://github.com/mephistofox/fxtun.dev/commit/7a76868ef953652f9bb7deb539a7d5a555aea92c))
* **web:** return number type from Input component for type=number ([4efcda1](https://github.com/mephistofox/fxtun.dev/commit/4efcda1cb10248388f5800962c888f0e7af9239e))

## [2.3.0](https://github.com/mephistofox/fxtun.dev/compare/v2.2.5...v2.3.0) (2026-02-04)


### Features

* **web:** add pricing section to landing page ([d5b3ccf](https://github.com/mephistofox/fxtun.dev/commit/d5b3ccf472ce0a4d2cabf4c42ee9be197bf6560c))


### Bug Fixes

* **web:** remove unused index variable in PricingSection ([19bad17](https://github.com/mephistofox/fxtun.dev/commit/19bad174b582c7a7cdecbc41a946a4bb58b055c2))

## [2.2.5](https://github.com/mephistofox/fxtun.dev/compare/v2.2.4...v2.2.5) (2026-02-03)


### Bug Fixes

* **ci:** fetch full git history in deploy workflow for correct version tags ([6899468](https://github.com/mephistofox/fxtun.dev/commit/689946876f1748ff6c6c750f2356e04771174ecc))

## [2.2.4](https://github.com/mephistofox/fxtun.dev/compare/v2.2.3...v2.2.4) (2026-02-03)


### Bug Fixes

* **server:** stream full request body instead of truncating to 256KB ([831c434](https://github.com/mephistofox/fxtun.dev/commit/831c434e12927acee962c5341b0de1f06888bf19))

## [2.2.3](https://github.com/mephistofox/fxtun.dev/compare/v2.2.2...v2.2.3) (2026-02-03)


### Bug Fixes

* **server:** show interstitial only for HTML responses ([36e8abb](https://github.com/mephistofox/fxtun.dev/commit/36e8abbd9165770a2613a78e90d0aa800d9c3fb3))

## [2.2.2](https://github.com/mephistofox/fxtun.dev/compare/v2.2.1...v2.2.2) (2026-02-02)


### Bug Fixes

* **client:** use UDP socket for local proxy in UDP tunnels ([daaf93e](https://github.com/mephistofox/fxtun.dev/commit/daaf93e26d7954110213f9a383487dca9512b4c5))

## [2.2.1](https://github.com/mephistofox/fxtun.dev/compare/v2.2.0...v2.2.1) (2026-02-02)


### Bug Fixes

* **client:** create temp file in same dir as binary to avoid cross-device rename ([41173e2](https://github.com/mephistofox/fxtun.dev/commit/41173e27400b3703283cab894a0a6ad46a059351))

## [2.2.0](https://github.com/mephistofox/fxtun.dev/compare/v2.1.0...v2.2.0) (2026-02-02)


### Features

* **cli:** add up/status/down commands and daemon-aware tunnel creation ([65d9d08](https://github.com/mephistofox/fxtun.dev/commit/65d9d082f765a232cb2886c56868bbb1e300712e))
* **daemon:** add ClientManager adapter for tunnel management ([475a6fd](https://github.com/mephistofox/fxtun.dev/commit/475a6fd37fee4ca8f7a94e2d8462f48066fa4d9d))
* **daemon:** add local HTTP API for daemon IPC ([ba893d4](https://github.com/mephistofox/fxtun.dev/commit/ba893d4fc591d5fce90c5de390200e38abe3d948))
* **daemon:** add process liveness and daemon running check ([d0008c5](https://github.com/mephistofox/fxtun.dev/commit/d0008c5ca04b1930d579dbf58f4d0a7e9a9731dc))
* **daemon:** add state file save/load helpers ([10cbae5](https://github.com/mephistofox/fxtun.dev/commit/10cbae505b36b9907fcb16509c4d0328a8a59cd5))


### Bug Fixes

* **daemon:** fix lint issues — file perms, ReadHeaderTimeout, errcheck ([fe17938](https://github.com/mephistofox/fxtun.dev/commit/fe17938a7e0f9df9331be67716ae092e4bd1c3f7))

## [2.1.0](https://github.com/mephistofox/fxtun.dev/compare/v2.0.0...v2.1.0) (2026-02-02)


### Features

* **api:** add plan management endpoints and enforce plan limits ([ef96a0d](https://github.com/mephistofox/fxtun.dev/commit/ef96a0d43fe2f6ef64a82d4885e1cd0fd3dd7869))
* **cli:** add --subdomain alias for --domain on http command ([f3c229e](https://github.com/mephistofox/fxtun.dev/commit/f3c229e2fa36d4a244803861d720121e1f67f6cc))
* **gui:** add GitHub OAuth authentication flow ([ca80658](https://github.com/mephistofox/fxtun.dev/commit/ca80658d0ee28ceede40c9f35c7893274d580927))
* **gui:** add use-case template cards to dashboard empty state ([7a1ea10](https://github.com/mephistofox/fxtun.dev/commit/7a1ea1086a18e1fc9d7770acdc6713ef24ad4018))
* **server:** add plans system with per-user limits ([bf34366](https://github.com/mephistofox/fxtun.dev/commit/bf34366470da52cda1c1d92d4dd7f2b9304ec043))
* **web:** add plan support, redesign profile, fix limits display ([06c1a59](https://github.com/mephistofox/fxtun.dev/commit/06c1a59d2f444e2f093bfcefe27e5e189215d1e4))
* **web:** improve dashboard, tokens, domains, and downloads UX ([672e709](https://github.com/mephistofox/fxtun.dev/commit/672e7098da1e7ebad8e844720ac870bd6655c2f7))
* **web:** redesign all admin views with filters and full i18n ([3754bfc](https://github.com/mephistofox/fxtun.dev/commit/3754bfcfc4d741d99cdb612f433b50825570f19c))
* **web:** show email instead of phone in admin users table ([7f44996](https://github.com/mephistofox/fxtun.dev/commit/7f44996b28bda715c99cd8d8991e335a3f536087))


### Bug Fixes

* **server:** fix embed directive and OAuth phone field ([6cde91e](https://github.com/mephistofox/fxtun.dev/commit/6cde91e6b11820df35638b9c620c3bd5197ea00e))
* **web:** show blocked account message on login and fix TOTP code check ([2c76c2b](https://github.com/mephistofox/fxtun.dev/commit/2c76c2b348dcafbb4ecfd78d87b69fc8add93717))

## [2.0.0](https://github.com/mephistofox/fxtun.dev/compare/v1.19.1...v2.0.0) (2026-02-01)


### ⚠ BREAKING CHANGES

* protocol and auth changes since v1.17 make older clients incompatible with the server.

### Features

* bump to v2.0 — no backward compatibility with pre-1.17 clients ([8f86e75](https://github.com/mephistofox/fxtun.dev/commit/8f86e752716c543b3ba8db3189dd05d7ccf70f62))
* **client:** add forced auto-update when client version is below server min_version ([07b6c15](https://github.com/mephistofox/fxtun.dev/commit/07b6c15bfa1ab523257da1375f6adfaa9aa64257))


### Bug Fixes

* **build:** use nearest semver tag for VERSION instead of git hash ([ba5c767](https://github.com/mephistofox/fxtun.dev/commit/ba5c767312745ac7c543643c8564598eb16f6177))
* **client:** use HTTPS on standard port for update check instead of control port ([371494d](https://github.com/mephistofox/fxtun.dev/commit/371494d966be8ea462f4315b921d9a3b81255e42))
* **lint:** suppress G204 gosec warning for syscall.Exec in self-restart ([d6c4b1c](https://github.com/mephistofox/fxtun.dev/commit/d6c4b1c690f95a78a805dbf42bddd99936d9666f))

## [1.19.1](https://github.com/mephistofox/fxtun.dev/compare/v1.19.0...v1.19.1) (2026-02-01)


### Bug Fixes

* **auth:** fix OAuth registration and support email-based login ([101e944](https://github.com/mephistofox/fxtun.dev/commit/101e9444895bd57bcab41debc5b52fab956691d5))

## [1.19.0](https://github.com/mephistofox/fxtun.dev/compare/v1.18.0...v1.19.0) (2026-02-01)


### Features

* add multi-session connection pooling with binary stream headers ([8c51490](https://github.com/mephistofox/fxtun.dev/commit/8c514901c8ab25a58f855a1fc4110b5f9f5e5367))
* **client:** pretty CLI output with HTTP request logging ([4a84c12](https://github.com/mephistofox/fxtun.dev/commit/4a84c12e4380a38c6cee599d1e838c87630cf3fb))


### Bug Fixes

* resolve lint errors in import order and integer conversion ([4260b21](https://github.com/mephistofox/fxtun.dev/commit/4260b21b90e7e5a41c881f53250806d0c8d3b7f9))
* **test:** update default server address in config test ([c84e793](https://github.com/mephistofox/fxtun.dev/commit/c84e7938abb61b49bd6e3d25bc43c052578d5a4b))


### Reverts

* remove QUIC transport, restore yamux-only operation ([02b2246](https://github.com/mephistofox/fxtun.dev/commit/02b22467684e5b64e9b363a979f30498bf56de75))

## [1.18.0](https://github.com/mephistofox/fxtun.dev/compare/v1.17.1...v1.18.0) (2026-02-01)


### Features

* **admin:** add user merge and password reset functionality ([430138a](https://github.com/mephistofox/fxtun.dev/commit/430138a814c98bf57d3620cf8391ce1802a17463))
* **client:** add QUIC transport with automatic fallback to yamux ([88f8e9f](https://github.com/mephistofox/fxtun.dev/commit/88f8e9ff7f35f5048aa25f70c174500e9d0b3cce))
* **oauth:** add Google OAuth as second provider alongside GitHub ([35fc1bd](https://github.com/mephistofox/fxtun.dev/commit/35fc1bd53bebfd662e3514ceb3246a4b0bc631c1))
* **server:** add QUIC listener alongside TCP/yamux ([50b4d99](https://github.com/mephistofox/fxtun.dev/commit/50b4d994479fa95b40bffa947a980360f9e9a1d8))
* **transport:** add multiplexed transport abstraction interfaces ([0da2cf1](https://github.com/mephistofox/fxtun.dev/commit/0da2cf1bb255ac11912e9232b6f366bf5ba2a58f))
* **transport:** add QUIC adapter implementing transport.Session ([2d33c0d](https://github.com/mephistofox/fxtun.dev/commit/2d33c0d75ce42c9c86ca37aeac06cee16137d9a8))
* **transport:** add yamux adapter implementing transport.Session ([90c8178](https://github.com/mephistofox/fxtun.dev/commit/90c8178f9a42290a2320efaead424ff532a74a0d))


### Bug Fixes

* **lint:** suppress gosec G402 in test TLS configs ([e0745e0](https://github.com/mephistofox/fxtun.dev/commit/e0745e0fd9ba4fb8f39b58f9dcb117c38e8cfcb4))
* **oauth:** include google_id in UserDTO API response ([08fddbf](https://github.com/mephistofox/fxtun.dev/commit/08fddbf3ac18d0a5b706aa18a43c8d99742dd95f))
* **server:** start QUIC listener independently of tls.enabled flag ([37f05ea](https://github.com/mephistofox/fxtun.dev/commit/37f05ea4c3099456a663d6b85b06ba1e4e2010d9))

## [1.17.1](https://github.com/mephistofox/fxtun.dev/compare/v1.17.0...v1.17.1) (2026-01-31)


### Bug Fixes

* restore build/appicon.png accidentally deleted in e006978 ([0734360](https://github.com/mephistofox/fxtun.dev/commit/0734360189bfc5e6a2a5cf360ff45b22d13b2a90))

## [1.17.0](https://github.com/mephistofox/fxtun.dev/compare/v1.16.0...v1.17.0) (2026-01-31)


### Features

* **api:** add device flow endpoints for CLI browser auth ([9b05c2b](https://github.com/mephistofox/fxtun.dev/commit/9b05c2b406d1c3aa0146eaccb9a5a0e6f8cf6ea8))
* **api:** add GitHub OAuth login, register, and account linking ([c4c356d](https://github.com/mephistofox/fxtun.dev/commit/c4c356da0bbc0b84422f27b33c3f61977abb76a3))
* **auth:** add OAuth register/login and GitHub linking ([5a1b099](https://github.com/mephistofox/fxtun.dev/commit/5a1b0992d977d1f24c54749cf396e4f5dd76e4c5))
* **cli:** add 'domains' command for subdomain management ([e006978](https://github.com/mephistofox/fxtun.dev/commit/e006978940d3e1bed12c8bd2a70259c9e0145f96))
* **cli:** add browser-based device flow to login command ([6a51afb](https://github.com/mephistofox/fxtun.dev/commit/6a51afba2f00eb257d2ec971a892051ae9156934))
* **cli:** add checkAuth helper for keyring and home config ([8945bb2](https://github.com/mephistofox/fxtun.dev/commit/8945bb2afeaa6f97ac20421ebc1a07828a04adf9))
* **cli:** add compile-time DefaultServerURL variable ([361e8f6](https://github.com/mephistofox/fxtun.dev/commit/361e8f652c33f0b17a39e51f47904d51ef42e257))
* **cli:** add custom domains management to 'domains' command ([6cda1bf](https://github.com/mephistofox/fxtun.dev/commit/6cda1bf8e2d1ac1500c53e5072a9f1cbd7ba0e6d))
* **cli:** add interactive 'fxtunnel init' command ([bd610fd](https://github.com/mephistofox/fxtun.dev/commit/bd610fd19c72017ea8503e6db0e48f3523f64815))
* **config:** add OAuth settings for GitHub ([6fe35ea](https://github.com/mephistofox/fxtun.dev/commit/6fe35ea235d0d4e124901130480a8d0df66ff82b))
* **config:** prioritize fxtunnel.yaml over client.yaml in CWD ([1b49eb8](https://github.com/mephistofox/fxtun.dev/commit/1b49eb88b1ba1b67f96773125b0ec1103b6d716f))
* **db:** add OAuth fields to users table ([d7e88b8](https://github.com/mephistofox/fxtun.dev/commit/d7e88b8f24f075b9a517332346823994970f3c73))
* **db:** add OAuth user repository methods ([54e83f6](https://github.com/mephistofox/fxtun.dev/commit/54e83f66309f74cd7293268d876953d2fed677a4))
* **web:** add CLI auth confirmation page for device flow ([07585a1](https://github.com/mephistofox/fxtun.dev/commit/07585a11b9128cfab826f0c2281608dbaaf763dc))
* **web:** add GitHub account linking to profile page ([d945d0e](https://github.com/mephistofox/fxtun.dev/commit/d945d0e61821e94efe5d10df39ecab8272d8cec9))
* **web:** add GitHub OAuth button to login and register pages ([781484a](https://github.com/mephistofox/fxtun.dev/commit/781484a7edba3112b9f1cf13898e2421630cda53))
* **web:** add OAuth callback page ([fcdbc3e](https://github.com/mephistofox/fxtun.dev/commit/fcdbc3e81aad7ce504826b307fbe854b378a146a))
* **web:** redesign auth pages with GitHub as primary login method ([07836fb](https://github.com/mephistofox/fxtun.dev/commit/07836fbec6c04aa135b7aa0137e0ec09de9c87b5))


### Bug Fixes

* **config:** add yaml tags to TunnelConfig for correct serialization ([06ca50c](https://github.com/mephistofox/fxtun.dev/commit/06ca50c7d80f1972837a45df28fb611a51cf893d))
* resolve golangci-lint errors (errcheck, gosec) ([951fc14](https://github.com/mephistofox/fxtun.dev/commit/951fc14322f8d4c0343e8522e1ef6c83d3327427))

## [1.16.0](https://github.com/mephistofox/fxtun.dev/compare/v1.15.0...v1.16.0) (2026-01-31)


### Features

* add graceful shutdown, trace ID logging, auto-update and inspect replay ([8aad04a](https://github.com/mephistofox/fxtun.dev/commit/8aad04ae78244db22d3b9e9e0b44f89222fe4622))
* **landing:** add advanced features section with interactive demos ([c4cceee](https://github.com/mephistofox/fxtun.dev/commit/c4cceee2db3f0dbf88de07129beefc1ca4be871c))


### Bug Fixes

* **lint:** handle errcheck warnings in version comparison ([b12a828](https://github.com/mephistofox/fxtun.dev/commit/b12a828540ca386bac68ede0eb2cfc64a3fdf201))
* **security:** protect /metrics endpoint with auth middleware ([eb26ae3](https://github.com/mephistofox/fxtun.dev/commit/eb26ae3cfd6e25a221ad40a9d818ff514cd9b510))
* **tls:** fall back to ACME on-demand when cached cert is missing ([3747c90](https://github.com/mephistofox/fxtun.dev/commit/3747c903464ec21e5440db389ae71996e108aaff))

## [1.15.0](https://github.com/mephistofox/fxtun.dev/compare/v1.14.0...v1.15.0) (2026-01-30)


### Features

* **custom-domains:** support apex domains via A record and add DNS setup guide ([cac4614](https://github.com/mephistofox/fxtun.dev/commit/cac461449bb35aa24a891a0eb00c7c44766e1198))


### Bug Fixes

* **api:** add max password length, log JSON encoding errors ([b0206ab](https://github.com/mephistofox/fxtun.dev/commit/b0206aba5414b8bc1b0cdbdae00e29ed2529d650))
* **auth:** wrap registration in transaction to prevent invite code reuse ([2c66b4c](https://github.com/mephistofox/fxtun.dev/commit/2c66b4cab24dfa9714c55bbdfd996473c0cd1e8e))
* **ci:** set CGO_ENABLED=1 for server build to support SQLite ([3ce0ee7](https://github.com/mephistofox/fxtun.dev/commit/3ce0ee78cd6f2631d2f80d8e1dfb0dc84d092383))
* **config:** enforce minimum length for JWT secret and TOTP encryption key ([a370b74](https://github.com/mephistofox/fxtun.dev/commit/a370b746a5434b9b7f5fe50431e97b968786f93a))
* **docker:** add healthcheck and run as non-root user ([a2cbcf5](https://github.com/mephistofox/fxtun.dev/commit/a2cbcf51805123a7e890f9a1a1da16f88b05e1a3))
* **lint:** resolve errcheck and gosec warnings across codebase ([9c66b9c](https://github.com/mephistofox/fxtun.dev/commit/9c66b9c52be1d1a8e144d9a03261949c622431c8))

## [1.14.0](https://github.com/mephistofox/fxtun.dev/compare/v1.13.0...v1.14.0) (2026-01-30)


### Features

* add custom domains with CNAME binding and TLS certificates ([8288168](https://github.com/mephistofox/fxtun.dev/commit/8288168390b9225cdb3ee8c8d0a90e56a32b86e2))

## [1.13.0](https://github.com/mephistofox/fxtun.dev/compare/v1.12.2...v1.13.0) (2026-01-30)


### Features

* **server:** add WebSocket passthrough via connection hijacking ([ea91baf](https://github.com/mephistofox/fxtun.dev/commit/ea91baf97ad15c1ba6e80a988cb769f7b57ee9ea))

## [1.12.2](https://github.com/mephistofox/fxtun.dev/compare/v1.12.1...v1.12.2) (2026-01-30)


### Bug Fixes

* **server:** stream full response body when inspection is enabled ([fc7bd66](https://github.com/mephistofox/fxtun.dev/commit/fc7bd66bf56962de2b862a9914e53676ffc0696f))

## [1.12.1](https://github.com/mephistofox/fxtun.dev/compare/v1.12.0...v1.12.1) (2026-01-30)


### Bug Fixes

* **server:** make SO_REUSEPORT best-effort to prevent startup failures ([76caaa1](https://github.com/mephistofox/fxtun.dev/commit/76caaa12d3cdd3ac1fd8427db5c5db2b0f6ade25))

## [1.12.0](https://github.com/mephistofox/fxtun.dev/compare/v1.11.1...v1.12.0) (2026-01-30)


### Features

* **tunnel:** add stability and performance improvements ([1e95e79](https://github.com/mephistofox/fxtun.dev/commit/1e95e79d0a75e11e59231137ff3bec20fec0eff5))


### Bug Fixes

* remove unused limitedWriter and improve rate limiter cleanup ([c72abe0](https://github.com/mephistofox/fxtun.dev/commit/c72abe0a583de71126a3420066b9458007542496))

## [1.11.1](https://github.com/mephistofox/fxtun.dev/compare/v1.11.0...v1.11.1) (2026-01-30)


### Bug Fixes

* **gui:** add typed wails bindings for InspectService ([8f1d726](https://github.com/mephistofox/fxtun.dev/commit/8f1d726e57baef53ff1cb624e4711caaa5921c86))
* **gui:** correct wails import paths and add InspectService bindings ([acfc949](https://github.com/mephistofox/fxtun.dev/commit/acfc949969de6cc6cef5f153b4a4b9a6d1417ee3))
* **web:** use correct localStorage key for SSE auth token ([4fa1a80](https://github.com/mephistofox/fxtun.dev/commit/4fa1a80c7a68e1d5895692b44d4685a208c5ba6f))

## [1.11.0](https://github.com/mephistofox/fxtun.dev/compare/v1.10.2...v1.11.0) (2026-01-30)


### Features

* **api:** add inspect handlers (list, detail, clear, SSE stream) ([efa4863](https://github.com/mephistofox/fxtun.dev/commit/efa4863d2b7149cd3e25c8ba4681813d313f8fbe))
* **api:** add InspectProvider interface and inspect routes ([ce27c99](https://github.com/mephistofox/fxtun.dev/commit/ce27c9959c655731326207b6d60ad46144195d83))
* **config:** add inspect settings ([a611bb1](https://github.com/mephistofox/fxtun.dev/commit/a611bb1498b2f428e84153844624f133f76cd179))
* **gui:** add traffic inspection view with SSE streaming ([da2a494](https://github.com/mephistofox/fxtun.dev/commit/da2a4945a1051464a02ac28c71b6dfb0e497cd15))
* **inspect:** add CapturedExchange data model ([a6bb61b](https://github.com/mephistofox/fxtun.dev/commit/a6bb61ba042d1a71d7676cf0ec7d0dd474d2f36d))
* **inspect:** add Manager for per-tunnel buffers ([b577e7e](https://github.com/mephistofox/fxtun.dev/commit/b577e7eccc881f06ed833fc44fb0af948833cb3d))
* **inspect:** add RingBuffer with fan-out subscriptions ([5a56185](https://github.com/mephistofox/fxtun.dev/commit/5a56185f495b56af15ce86b1d84c4eb08461b0ca))
* **inspect:** capture HTTP traffic in HandleConnection ([5023c3b](https://github.com/mephistofox/fxtun.dev/commit/5023c3bc2cb5f86ea8596dcef70edc77b624f1fc))
* **server:** integrate InspectManager into server lifecycle ([45193c8](https://github.com/mephistofox/fxtun.dev/commit/45193c84289a1a189f66cf702c7aaadf03535f9e))
* **server:** wire InspectManager into API server ([83ef74f](https://github.com/mephistofox/fxtun.dev/commit/83ef74fa110fd8015f24600f0249105be14c33f7))
* **web:** add traffic inspection UI with real-time SSE ([dffe5c2](https://github.com/mephistofox/fxtun.dev/commit/dffe5c24085f6aec6b27e2ece6aadf6bc7cd1615))

## [1.10.2](https://github.com/mephistofox/fxtun.dev/compare/v1.10.1...v1.10.2) (2026-01-30)


### Performance Improvements

* **server:** add yamux stream pool for low-latency connection handling ([d16564a](https://github.com/mephistofox/fxtun.dev/commit/d16564ab9e802498024fd2795e6fbffbdff2d58e))

## [1.10.1](https://github.com/mephistofox/fxtun.dev/compare/v1.10.0...v1.10.1) (2026-01-30)


### Bug Fixes

* **lint:** use pointer types in sync.Pool, simplify select to sleep ([06c30c2](https://github.com/mephistofox/fxtun.dev/commit/06c30c22903d31b0c7f69622b9d0359ffdb75570))


### Performance Improvements

* **proxy:** optimize tunnel proxying for minimal overhead ([3e87fcb](https://github.com/mephistofox/fxtun.dev/commit/3e87fcb4c0368cab0ba4b75d732dc942ab2d5006))

## [1.10.0](https://github.com/mephistofox/fxtun.dev/compare/v1.9.1...v1.10.0) (2026-01-30)


### Features

* **cli:** rename --subdomain to --domain, add login/logout commands ([89f7666](https://github.com/mephistofox/fxtun.dev/commit/89f766651d10e5d43dd658cb16392b74219a48f4))

## [1.9.1](https://github.com/mephistofox/fxtun.dev/compare/v1.9.0...v1.9.1) (2026-01-29)


### Bug Fixes

* **ci:** filter artifact download to exclude docker buildx internals ([fa4ac74](https://github.com/mephistofox/fxtun.dev/commit/fa4ac7437e4a7cf7194b7569faecf852121b5d40))

## [1.9.0](https://github.com/mephistofox/fxtun.dev/compare/v1.8.3...v1.9.0) (2026-01-29)


### Features

* **server:** add i18n, embed templates, admin bypass for interstitial ([03b49dc](https://github.com/mephistofox/fxtun.dev/commit/03b49dc7c0b86b974e8e20549c6be6010a6af4d8))

## [1.8.3](https://github.com/mephistofox/fxtun.dev/compare/v1.8.2...v1.8.3) (2026-01-29)


### Bug Fixes

* **client:** wait for goroutines before reconnect to prevent WaitGroup panic ([1d0e177](https://github.com/mephistofox/fxtun.dev/commit/1d0e1771fae16382aa9ac84b22bf2f865af998d6))

## [1.8.2](https://github.com/mephistofox/fxtun.dev/compare/v1.8.1...v1.8.2) (2026-01-29)


### Bug Fixes

* **docker:** bump Go image to 1.24 to match go.mod requirement ([4239bcb](https://github.com/mephistofox/fxtun.dev/commit/4239bcb3022695f4efe362794eec7efd8609cd28))

## [1.8.1](https://github.com/mephistofox/fxtun.dev/compare/v1.8.0...v1.8.1) (2026-01-29)


### Bug Fixes

* **ci:** lowercase docker image name for ghcr.io compatibility ([5fc66f9](https://github.com/mephistofox/fxtun.dev/commit/5fc66f98621c377ea6f78cd24d6766afdbe95a62))

## [1.8.0](https://github.com/mephistofox/fxtun.dev/compare/v1.7.0...v1.8.0) (2026-01-29)


### Features

* **server:** add interstitial warning page for HTTP tunnels ([5f45ec3](https://github.com/mephistofox/fxtun.dev/commit/5f45ec379662d6734192a96cec0e5fe9675b83fc))


### Bug Fixes

* **gui:** disable systray on Windows to prevent window closing after auth ([25e06c5](https://github.com/mephistofox/fxtun.dev/commit/25e06c5b536dc0fb4e78af5f874d386706b2e6c1))
* **web:** use mfdev.ru/install.sh instead of get.mfdev.ru subdomain ([16bf098](https://github.com/mephistofox/fxtun.dev/commit/16bf098187b3d883698b9e6be8e0610aea1fea25))

## [1.7.0](https://github.com/mephistofox/fxtun.dev/compare/v1.6.0...v1.7.0) (2026-01-29)


### Features

* add IP whitelist on API tokens for control plane auth ([a5bb9fc](https://github.com/mephistofox/fxtun.dev/commit/a5bb9fc0deeb71818b3ee986e4ef61c737d50a23))
* add Prometheus metrics endpoint and request instrumentation ([74511e7](https://github.com/mephistofox/fxtun.dev/commit/74511e70246669dc3e157da9e4161d2c5937f218))
* add real-time traffic stats for tunnels ([984c468](https://github.com/mephistofox/fxtun.dev/commit/984c4689f5b77ec544d7d6f86bdfeed1d1f2680a))
* **gui:** add auto token refresh, blocked user screen, and support link ([117ce91](https://github.com/mephistofox/fxtun.dev/commit/117ce9127627a7d10b42359dc684fdb62ae5c344))
* **gui:** add system tray, log forwarding, build info, and auto-connect ([fa37bf7](https://github.com/mephistofox/fxtun.dev/commit/fa37bf7f077e551a3d03cba8596a545e80d65a6e))
* **security:** add per-IP rate limiting for API endpoints ([a0a5876](https://github.com/mephistofox/fxtun.dev/commit/a0a58762f8bb5b8d165264e2d844d9152878ea74))
* **security:** add security response headers middleware ([bc6c3ca](https://github.com/mephistofox/fxtun.dev/commit/bc6c3ca3b3313d9a4cb4832ccfe8cc8c58bbbe9c))
* **security:** replace wildcard CORS with configurable origins ([490d77f](https://github.com/mephistofox/fxtun.dev/commit/490d77fc856a417fb6d269676d7dcc220fc6ef6b))
* **security:** require jwt_secret and totp_key when web panel is enabled ([a7ca5e2](https://github.com/mephistofox/fxtun.dev/commit/a7ca5e2a08720b1a28a8e4163ac7e3f3ec7dadb0))
* **web:** add dynamic version display and i18n additions ([4e87e1c](https://github.com/mephistofox/fxtun.dev/commit/4e87e1c27e75a575eac245bb867eece72887c885))
* **web:** add install script, download section with platform picker, and update domains ([a0f220a](https://github.com/mephistofox/fxtun.dev/commit/a0f220a73e12e6ca9c24812953812cbadcb11fce))


### Bug Fixes

* **build:** pass ldflags to wails dev and build commands ([5a00d18](https://github.com/mephistofox/fxtun.dev/commit/5a00d182d0b1b8736fda58fe64beb28fdaa342ae))
* **ci:** add BuildTime to GUI ldflags and update Go to 1.24 ([9e963eb](https://github.com/mephistofox/fxtun.dev/commit/9e963ebb56d35441e24a4c05f51411383ad11e84))
* **gui:** improve layout compactness and fix UI issues ([33fc4de](https://github.com/mephistofox/fxtun.dev/commit/33fc4de13efe2bcd34ddd2e1d8f2282f9d04c4cf))
* **gui:** use format string in fmt.Errorf calls ([d2c0ad4](https://github.com/mephistofox/fxtun.dev/commit/d2c0ad4788f77a5fc09231326ce82da16e826bf2))
* resolve critical race conditions, goroutine leaks, and performance issues ([2832875](https://github.com/mephistofox/fxtun.dev/commit/2832875f17a9436897623682c47cf763c0d2847e))
* resolve golangci-lint errors and tidy go.mod ([af266ba](https://github.com/mephistofox/fxtun.dev/commit/af266ba3698f226c0e8d776832b4a6e9d6158608))
* **web:** fix mobile overflow issues on landing page ([2050f9c](https://github.com/mephistofox/fxtun.dev/commit/2050f9cb37d96931b41ef697964e8e6081628b55))
* **web:** reduce section spacing on mobile and hide Learn More button ([0dd32c3](https://github.com/mephistofox/fxtun.dev/commit/0dd32c3bea73676e0522ca0e1bd119e93a78a4e8))
* **web:** show nav background when mobile menu is open at top ([2ec6bf5](https://github.com/mephistofox/fxtun.dev/commit/2ec6bf56557852ca5eae28463b2bebc968e2eefa))


### Performance Improvements

* **client:** cache resolved local address to skip IPv4/IPv6 probe ([f282bf7](https://github.com/mephistofox/fxtun.dev/commit/f282bf77d940f59f5040e7d0e116dc74c8505583))
* **client:** race IPv4/IPv6 in parallel and pre-probe on tunnel creation ([875df70](https://github.com/mephistofox/fxtun.dev/commit/875df7025151b8ebae4b83fbf6f440e33c2ec490))

## [1.6.0](https://github.com/mephistofox/fxtun.dev/compare/v1.5.0...v1.6.0) (2026-01-28)


### Features

* **gui:** complete redesign with cyber-industrial theme ([39bb251](https://github.com/mephistofox/fxtun.dev/commit/39bb251628e3f2b40d963dced4f5810d752a8256))

## [1.5.0](https://github.com/mephistofox/fxtun.dev/compare/v1.4.0...v1.5.0) (2026-01-28)


### Features

* **gui:** redesign to cyber-industrial theme ([758fe35](https://github.com/mephistofox/fxtun.dev/commit/758fe35f4bea1b7feb63431f984bc399b9c7377a))


### Bug Fixes

* **server:** prevent 404 page layout shift on mobile ([4109f81](https://github.com/mephistofox/fxtun.dev/commit/4109f81b3bfc06dc4967b0f0b980559888a12029))

## [1.4.0](https://github.com/mephistofox/fxtun.dev/compare/v1.3.1...v1.4.0) (2026-01-27)


### Features

* **client:** add automatic JWT token refresh on reconnect ([1feff99](https://github.com/mephistofox/fxtun.dev/commit/1feff9990a6bd0c937e0241c30e53f72acb752fd))
* **server:** add custom styled 404 error page ([2b5d08a](https://github.com/mephistofox/fxtun.dev/commit/2b5d08ab24d122fc507b22d9d2b520dd87a9a177))
* **web:** redesign landing page with modern cyber-industrial theme ([2f76e15](https://github.com/mephistofox/fxtun.dev/commit/2f76e15a8791382c168985386d56e3e428ee751c))
* **web:** update landing page fonts and add animated card borders ([fdcb6d7](https://github.com/mephistofox/fxtun.dev/commit/fdcb6d7d5712ddb524a30af5c6baf8d410c2633e))

## [1.3.1](https://github.com/mephistofox/fxtun.dev/compare/v1.3.0...v1.3.1) (2026-01-26)


### Bug Fixes

* **build:** clean old files before building to downloads/ ([667e278](https://github.com/mephistofox/fxtun.dev/commit/667e278f9dc8201d4fffd4fd2dfa1c89b0855788))

## [1.3.0](https://github.com/mephistofox/fxtun.dev/compare/v1.2.0...v1.3.0) (2026-01-26)


### Features

* **gui:** add refresh token support for persistent sessions ([c3b6eba](https://github.com/mephistofox/fxtun.dev/commit/c3b6eba8ff894e4c2cc9d96f24f9750520b91c39))


### Bug Fixes

* **ci:** make downloads sync non-fatal when assets not ready yet ([f0da8f4](https://github.com/mephistofox/fxtun.dev/commit/f0da8f492d666eff63f86f43f667ad800b3fc520))
* **gui:** use token method for auto-login to fix session persistence ([04e64fa](https://github.com/mephistofox/fxtun.dev/commit/04e64fabcd4df1ffcb239b5498cdda8d0f2f327f))

## [1.2.0](https://github.com/mephistofox/fxtun.dev/compare/v1.1.1...v1.2.0) (2026-01-26)


### Features

* **ci:** sync client downloads from latest release on deploy ([ed3ef19](https://github.com/mephistofox/fxtun.dev/commit/ed3ef19ac789ca7a79467338e9bc7f8f66f65cdf))


### Bug Fixes

* **ci:** remove conflicting --skip-existing flag from gh release download ([6de8a31](https://github.com/mephistofox/fxtun.dev/commit/6de8a31221c3814a8ed7c19a09eed79d54225615))

## [1.1.1](https://github.com/mephistofox/fxtun.dev/compare/v1.1.0...v1.1.1) (2026-01-26)


### Bug Fixes

* **ci:** enable CGO for server build (required by go-sqlite3) ([88fce88](https://github.com/mephistofox/fxtun.dev/commit/88fce887dcc2f3327173f8553098861d2ffc5284))
* **ci:** use workflow_run trigger for deploy instead of wait-on-check ([62dd81a](https://github.com/mephistofox/fxtun.dev/commit/62dd81a436a2e7fecaab8b88c0784fdd6b96a497))
* **client:** add IPv4/IPv6 fallback for local service connections ([766ba87](https://github.com/mephistofox/fxtun.dev/commit/766ba871e1fb513bc2f1b2b5a7fc02a86877d4a3))

## [1.1.0](https://github.com/mephistofox/fxtun.dev/compare/v1.0.0...v1.1.0) (2025-12-25)


### Features

* **admin:** add admin panel with tunnels, audit logs and user management ([2c4bb1c](https://github.com/mephistofox/fxtun.dev/commit/2c4bb1cf3050ed6775081a926d5506ef1170f444))
* **gui,web:** add management panel and desktop application ([536e074](https://github.com/mephistofox/fxtun.dev/commit/536e07426333dedcd3eea4b2fccb6b5c71b88a22))
* initial implementation of fxTunnel reverse tunneling system ([9e851ab](https://github.com/mephistofox/fxtun.dev/commit/9e851ab38d60d94ed9b15e80f796b7f61a909bfd))
* **sync:** add data synchronization between GUI and server ([0812d4c](https://github.com/mephistofox/fxtun.dev/commit/0812d4cc3b16825cef9fba786749a35fcb7a141f))


### Bug Fixes

* **ci:** build frontends before tests and lint ([fc81616](https://github.com/mephistofox/fxtun.dev/commit/fc81616b65c7cefb9609674be70a500771afcd1f))
* **ci:** disable errcheck linter, use exclude-dirs ([8b30028](https://github.com/mephistofox/fxtun.dev/commit/8b300288d08a56c09ff509c75126171d8927545b))
* **ci:** exclude GUI from CI checks, use placeholder dist ([4a3d7aa](https://github.com/mephistofox/fxtun.dev/commit/4a3d7aad6735c653d171bd1200cbfeb9cd6a6b01))
* **ci:** explicitly disable errcheck linter ([c3359c7](https://github.com/mephistofox/fxtun.dev/commit/c3359c7e4d869da864986ca8f86b9244bd9af190))
* **ci:** handle different Wails output paths on macOS ([24ecb86](https://github.com/mephistofox/fxtun.dev/commit/24ecb86b709742186b3d37938b5070e7465b8008))
* remove unused variables and apply gosimple suggestion ([03da85c](https://github.com/mephistofox/fxtun.dev/commit/03da85c25a78b18a24191cd887b2f1510f9b878d))

## 1.0.0 (2025-12-25)


### Features

* **admin:** add admin panel with tunnels, audit logs and user management ([2c4bb1c](https://github.com/mephistofox/fxtun.dev/commit/2c4bb1cf3050ed6775081a926d5506ef1170f444))
* **gui,web:** add management panel and desktop application ([536e074](https://github.com/mephistofox/fxtun.dev/commit/536e07426333dedcd3eea4b2fccb6b5c71b88a22))
* initial implementation of fxTunnel reverse tunneling system ([9e851ab](https://github.com/mephistofox/fxtun.dev/commit/9e851ab38d60d94ed9b15e80f796b7f61a909bfd))
* **sync:** add data synchronization between GUI and server ([0812d4c](https://github.com/mephistofox/fxtun.dev/commit/0812d4cc3b16825cef9fba786749a35fcb7a141f))


### Bug Fixes

* **ci:** build frontends before tests and lint ([fc81616](https://github.com/mephistofox/fxtun.dev/commit/fc81616b65c7cefb9609674be70a500771afcd1f))
* **ci:** disable errcheck linter, use exclude-dirs ([8b30028](https://github.com/mephistofox/fxtun.dev/commit/8b300288d08a56c09ff509c75126171d8927545b))
* **ci:** exclude GUI from CI checks, use placeholder dist ([4a3d7aa](https://github.com/mephistofox/fxtun.dev/commit/4a3d7aad6735c653d171bd1200cbfeb9cd6a6b01))
* **ci:** explicitly disable errcheck linter ([c3359c7](https://github.com/mephistofox/fxtun.dev/commit/c3359c7e4d869da864986ca8f86b9244bd9af190))
* **ci:** handle different Wails output paths on macOS ([24ecb86](https://github.com/mephistofox/fxtun.dev/commit/24ecb86b709742186b3d37938b5070e7465b8008))
* remove unused variables and apply gosimple suggestion ([03da85c](https://github.com/mephistofox/fxtun.dev/commit/03da85c25a78b18a24191cd887b2f1510f9b878d))

## 1.0.0 (2025-12-25)


### Features

* **admin:** add admin panel with tunnels, audit logs and user management ([8f4ac55](https://github.com/mephistofox/fxtun.dev/commit/8f4ac55d5de0b533515c74814e248259fc26bd96))
* **gui,web:** add management panel and desktop application ([536e074](https://github.com/mephistofox/fxtun.dev/commit/536e07426333dedcd3eea4b2fccb6b5c71b88a22))
* initial implementation of fxTunnel reverse tunneling system ([9e851ab](https://github.com/mephistofox/fxtun.dev/commit/9e851ab38d60d94ed9b15e80f796b7f61a909bfd))
* **sync:** add data synchronization between GUI and server ([af415a4](https://github.com/mephistofox/fxtun.dev/commit/af415a48959b40574a8d801443053d05a07682c1))


### Bug Fixes

* **ci:** build frontends before tests and lint ([7cf0ff9](https://github.com/mephistofox/fxtun.dev/commit/7cf0ff9e15c563c15f3476b719db429447d08fd0))
* **ci:** disable errcheck linter, use exclude-dirs ([e0d1af8](https://github.com/mephistofox/fxtun.dev/commit/e0d1af8677d8a0e136b265eeaa9016e9adf58bf8))
* **ci:** exclude GUI from CI checks, use placeholder dist ([4927b8c](https://github.com/mephistofox/fxtun.dev/commit/4927b8c3e34b0ce73190f81e211a7831901f7996))
* **ci:** explicitly disable errcheck linter ([4485f22](https://github.com/mephistofox/fxtun.dev/commit/4485f223e4449267d604eb1a62aaae97d2a8d075))
* remove unused variables and apply gosimple suggestion ([b652e8b](https://github.com/mephistofox/fxtun.dev/commit/b652e8b03a858f0b54eb21c9612393e4f695be09))
