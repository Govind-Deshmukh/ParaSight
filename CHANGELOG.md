# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2025-01-31

### Added
- Initial release
- Pull-based log tailing via HTTP endpoints
- System metrics: CPU, memory (RAM + swap), disk
- Configurable log endpoints with `?lines=` parameter (default 20, max 100)
- Health check endpoint
- Skips NAS/NFS mounts automatically
- Timestamp on all metric responses for time series collection
