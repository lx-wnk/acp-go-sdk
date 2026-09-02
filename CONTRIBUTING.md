# Contributing

## Upstream first

This repository is a fork of
[coder/acp-go-sdk](https://github.com/coder/acp-go-sdk), maintained because
upstream review has stalled. It is a holding position, not a split.

If your change is a general fix rather than something specific to this fork,
please also open it against `coder/acp-go-sdk`, or say in your pull request
why that is not possible yet. When upstream picks the work back up, this fork
offers its patches back.

## Generated code is never hand-edited

`agent_gen.go`, `client_gen.go`, `constants_gen.go`, `helpers_gen.go` and
`types_gen.go` are produced by `cmd/generate` from the ACP JSON schema, and
they reproduce byte-identically on every run.

A fix therefore belongs in the generator or in the schema input, never in the
generated file — a hand-patch disappears on the next regeneration. Regenerate
and commit the result together with the generator change:

```bash
make version
```

## Setup

The toolchain is provisioned with [mise](https://mise.jdx.dev):

```bash
mise install
```

## Verify before opening a pull request

```bash
mise exec -- make check   # treefmt, then `git diff --exit-code`
mise exec -- make test    # go test ./... and go build ./example/...
```

These are the exact two commands CI runs, so a change that fails either one
here fails there too. `golangci-lint` is provisioned by mise but is not wired
into `make check`, `make test` or CI — treat its output as advisory.

## Commits and pull requests

Commits follow [Conventional Commits](https://www.conventionalcommits.org/),
scoped to the area touched:

```
fix(generate): refuse an unrecognised discriminator
feat(generate): record the public API delta of a schema bump
chore(schema): bump ACP schema to 1.21.0
ci: run the pipeline on pushes to trunk
```

Pull requests are squash-merged, so the pull request title becomes the commit
subject — keep it in the same `type(scope): summary` form. One intent per pull
request; a stack of small ones reviews faster than a single large one.

## Releases

The module version tracks the ACP schema release it was generated from, not Go
API compatibility — see [Versioning](README.md#versioning) for what that means
for your code, and `RELEASING.md` for the release procedure itself.
