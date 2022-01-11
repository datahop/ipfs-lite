# Change Log
All notable changes to this project will be documented in this file.

## [v0.0.14] - 2022-1-11

### Added

- [0fca1cb](https://github.com/datahop/ipfs-lite/commit/0fca1cb) Allow custom encryption
- [450c508](https://github.com/datahop/ipfs-lite/commit/450c508) Gateway added

### Changed

- [207e4df](https://github.com/datahop/ipfs-lite/commit/207e4df) Start takes in options rather than booleans

## [v0.0.13] - 2021-11-27

### Added

- [6239e1e](https://github.com/datahop/ipfs-lite/commit/6239e1e) Optional passphrase based content encryption
- [6708b03](https://github.com/datahop/ipfs-lite/commit/6708b03) Support for private network

### Changed

- [df060c1](https://github.com/datahop/ipfs-lite/commit/df060c1) zeroconf mDNS replaces libp2p mDNS 
- [df060c1](https://github.com/datahop/ipfs-lite/commit/df060c1) D2D now uses zeroconf mDNS service instead of manually handing connections

## [v0.0.12] - 2021-10-07

### Added

- [655b263](https://github.com/datahop/ipfs-lite/commit/655b263) Auto Disconnect after content replication for D2D discovery

### Changed

- [84b7b42](https://github.com/datahop/ipfs-lite/commit/84b7b42) Do not initiate D2D Discovery in nodes are pre-connected 

## [v0.0.11] - 2021-09-22

### Added

- [6a69aa2](https://github.com/datahop/ipfs-lite/commit/6a69aa2) Content and connection Matrix
- [81b7146](https://github.com/datahop/ipfs-lite/commit/81b7146) Cli Documentation added
- [758a9f3](https://github.com/datahop/ipfs-lite/commit/758a9f3) Cli Client

## [v0.0.10] - 2021-07-14

First alpha release of Datahop mobile-client

### Added

- [4791115](https://github.com/datahop/ipfs-lite/commit/4791115) Optional Bootstrapping with datahop bootstrap node
- [c6ed293](https://github.com/datahop/ipfs-lite/commit/c6ed293) Use bloomfilter for Replication state keeping
- [6ccd96c](https://github.com/datahop/ipfs-lite/commit/6ccd96c) Content Indexing & Replication using [crdt](https://github.com/ipfs/go-ds-crdt)
- [840afc1](https://github.com/datahop/ipfs-lite/commit/840afc1) Discovery using BLE & Wifi-Direct
- [a15d8d6](https://github.com/datahop/ipfs-lite/commit/a15d8d6) mDNS Discovery for home networks
- [749ba58](https://github.com/datahop/ipfs-lite/commit/749ba58) Persistent repository & config
