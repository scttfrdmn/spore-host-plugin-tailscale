# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-03-22

### Added
- `tailscale/plugin.yaml`: install Tailscale and join a tailnet as an ephemeral node
- `--hostname` set to `{{ instance.name }}` (the spore's EC2 Name tag)
- `accept_dns` config option (default `false`) to control tailnet DNS on the instance
- `tailscale logout` in stop steps for clean ephemeral-node deregistration
- `tailscale status` health check (1-minute interval)

[Unreleased]: https://github.com/scttfrdmn/spore-host-plugin-tailscale/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/scttfrdmn/spore-host-plugin-tailscale/releases/tag/v0.1.0
