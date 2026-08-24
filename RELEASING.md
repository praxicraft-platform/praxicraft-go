# Releasing

Maintainer notes for publishing this module to the Go module proxy / pkg.go.dev. Integrators should ignore this file — see the [README](README.md).

## How publish works

Go modules do not use a private registry like PyPI/npm. A release is an annotated git tag that the [module proxy](https://proxy.golang.org) can fetch.

[`.github/workflows/publish.yml`](.github/workflows/publish.yml) runs on pushes to **`main`** when:

- `*.go`
- `go.mod`
- `CHANGELOG.md`
- `.github/workflows/publish.yml`

Flow:

1. Run `go test` on Go 1.22 / 1.23 / 1.24.
2. Read `Version` from `version.go`.
3. If git tag `v{version}` already exists → skip tagging.
4. Otherwise create + push annotated tag `v{version}`.

After the tag exists, pkg.go.dev indexes the module (may take a few minutes; you can also ping `https://proxy.golang.org/github.com/praxicraft-platform/praxicraft-go/@v/v{version}.info`).

## Cut a release

1. Bump `Version` in `version.go`.
2. Update `CHANGELOG.md`.
3. Merge to `main`.

## One-time setup

No registry credentials. Ensure the default branch is `main` and the `github-actions` bot can push tags (`contents: write` on the publish job).

## GitHub Release

The Publish workflow also creates a **GitHub Release** for tag `v{version}` (with generated notes and package assets where applicable).

You can run **Actions → Publish → Run workflow** manually (`workflow_dispatch`) after bumping the version on `main`.

## Auto-bump

Pushes to `main` that change package source auto-bump the patch version and `CHANGELOG.md` in the runner, create a local `chore(release)` commit, and **push only the annotated tag** (not `main` — branch protection requires PRs). The tag retains the release commit. Then CI creates a **GitHub Release** and publishes to the language registry when credentials are configured.

Version files on `main` may lag until a follow-up PR syncs them; published artifacts always use the tagged version.

Skip with `[skip release]` in the commit message.
