# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## 2026-01-22
FEATURE: Add "Inspiration" section to README crediting grepai as foundation
FEATURE: Auto-start PostgreSQL container on `agentdx watch` and `agentdx session start`
FEATURE: Add `--pg-name` and `--pg-port` flags to `watch` and `session start` commands
FEATURE: Add config file support for `container_name` and `port` under `index.store.postgres`
FEATURE: Add Docker volume support for persistent data across container restarts
FEATURE: Add TestContainer helper for parallel test execution with random containers

## 2026-01-21
FEATURE: Initial refactor  grepai from semantic to full text search
FIX: Ensure compose.yaml is generated even when Docker is unavailable (TestSetupPostgresBackend_NoDocker)
FIX: Suppress expected race condition log noise in TestPIDFile_AtomicWrite
FIX: Add Docker container cleanup with wait loop to prevent "container name already in use" errors in CI
FIX: Add TestMain functions to clean up stale Docker containers before running tests
FIX: Add retry logic in RunLocalSetup for concurrent container creation attempts

