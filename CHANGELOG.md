# Changelog

## 1.0.0

### Breaking

- Rename assessment task methods from `ListCases` / `AttachCases` / `ReplaceCases` / `RemoveCase` to `ListTasks` / `AttachTasks` / `ReplaceTasks` / `RemoveTask`.
- Wire paths use `/tasks/` instead of `/cases/`; JSON keys use `tasks`, `task_id`, and `assessment_task_id`.

## 0.1.1

- ci: auto-bump releases with GitHub Release + package publish
- Add GitHub Release publishing to the Publish workflow.
- Fix API reference link in README.md

## [0.1.0] — 2026-08-14

### Added

- Initial `praxicraft` Go SDK for the Assess Public API.
- `Client` with Bearer API-key auth (`PRAXICRAFT_API_KEY` / `PRAXICRAFT_API_BASE_URL`).
- Automatic retries on `429` / `5xx` / transport errors (default 2 retries), honouring numeric and HTTP-date `Retry-After`.
- Typed response structs (`Org`, `Assessment`, `Invite`, `ResultRow`, …) via resource methods / `DoAs`.
- Typed errors mapped from the Public API `{ "error": { "code", "message" } }` envelope.
- Resources: `Assessments` (list / retrieve / create / update / activate / cases), `Invites`, `Results`, `Org`, `Webhooks` (CRUD + test), `Pipelines` (list / enroll).
- Local webhook helper `VerifySignature` for `X-Praxicraft-Signature` (`nil` body = empty payload).
- CI on Go 1.22–1.24 and auto-tag publish on `main`.
