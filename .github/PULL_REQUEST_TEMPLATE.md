## Summary

<!-- What does this change do, and why? -->

## Upstream

- [ ] Fork-specific (schema version, fork identity, CI) — no upstream action needed.
- [ ] General fix, and also opened against
  [coder/acp-go-sdk](https://github.com/coder/acp-go-sdk), or will be once
  this merges. Link:

## Checklist

- [ ] `mise exec -- make check` passes
- [ ] `mise exec -- make test` passes
- [ ] No `*_gen.go` was hand-edited — regenerated with `make version` instead
- [ ] Title follows Conventional Commits (`type(scope): summary`)
