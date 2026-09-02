# Releasing

This project follows the ACP schema version published by
[`agentclientprotocol/agent-client-protocol`](https://github.com/agentclientprotocol/agent-client-protocol).
What that means for a consumer's code is described once, under
[Versioning](README.md#versioning); this file covers the release procedure only.

## Prerequisites

- `make`, `curl`, and `git` in your `PATH`.
- The toolchain, provisioned with `mise install` — see
  [AGENTS.md](AGENTS.md#build-test-and-development-commands).

## Bump the Schema Version

Usually you do not: `.github/workflows/schema-update.yaml` runs daily, resolves
the newest upstream schema tag, regenerates, and opens a pull request titled
`chore(schema): bump ACP schema to <version>` from a `schema/update-<version>`
branch. Check for that pull request before starting a bump by hand, or the two
will collide on the same branch name. The steps below are for an out-of-band
bump — pinning an older tag, or rerunning after the workflow failed.

1. Decide which upstream ACP schema tag to adopt (for example `v0.4.3`).
1. Update `schema/version` and regenerate code. There are two supported ways to
   do this:

### Option A: Use the release helper

```bash
make release VERSION=0.4.3
```

The helper performs the following steps:

- writes the requested number to `schema/version`
- runs `make version` to download the new schema files and regenerate Go code
- runs `make fmt`, `make test`, and `make check`
- asserts that `schema/version` and `version` now match

If any command fails, fix the issue and rerun the helper. The target does not
create commits or tags; it just prepares the tree.

### Option B: Run the steps manually

```bash
printf '0.4.3\n' > schema/version
make version
make fmt
GOCACHE=$(pwd)/.gocache make test
make check
cmp -s schema/version version
```

`make version` downloads the schema files for the requested ACP tag, regenerates
all Go code, and formats the repository with `gofumpt`. The `cmp` command is a
lightweight guard that ensures both `schema/version` and the top-level `version`
file agree before you publish.

## Review and Commit

1. Inspect the changes: `git status` and `git diff` should show updated schema
   files, generated Go code, and the version files.
1. Commit with a descriptive message such as `release: v0.4.3`.
1. Push the branch to GitHub and open a pull request if review is required.

## Tag and Publish

1. Tag the release commit with a Go-compatible tag:

   ```bash
   git tag v0.4.3
   git push origin v0.4.3
   ```

1. Create a GitHub release for the tag. Include a summary of notable changes and
   reference the upstream ACP schema version.

   Pass `--generate-notes` to have GitHub list the merged pull requests and a
   full-changelog link, then edit the body to lead with what the change means
   for consumers:

   ```bash
   gh release create v0.4.3 --generate-notes
   ```

The tag is what consumers pin, but not through `go get`: the module path is
upstream's, so `go get github.com/coder/acp-go-sdk@vX.Y.Z` resolves against
Coder's repository and cannot see a tag pushed here. Consumers reach this fork
through a `replace` directive naming the tag — see
[Installation](README.md#installation). Push the tag before announcing the
release, and make sure the version it names is the one the README's install
recipe prints.

## Additional Notes

- If the new schema introduces breaking changes, update examples and docs in
  the same commit.
- `CHANGELOG.md` is maintained by the generator: a schema bump records the
  exported identifiers it removed, changed and added, which is the part of a
  release consumers act on.
- The helper uses a repository-local Go build cache (`.gocache`) to avoid
  sandbox restrictions in CI and local development. You can delete it with
  `rm -rf .gocache` if needed.
- `make clean` removes the downloaded schema files and the `version` file; rerun
  the release steps afterwards if you invoke it.
