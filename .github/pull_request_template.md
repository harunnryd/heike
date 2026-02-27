## Summary

- What changed?
- Why is this change needed?

## Behavior and Call Chain Impact

- Entrypoint(s):
- Runtime wiring touched:
- Internal package(s) affected:

## Validation

Commands run:

```sh
./scripts/ci/agents_guard.sh
go test ./...
go build ./cmd/heike
go vet ./...
```

Result summary:

- [ ] All commands passed
- [ ] If anything failed, failure details are included

## Docs and Config

- [ ] Docs updated (`docs/`) for behavior/contract changes
- [ ] Config template/defaults updated when config behavior changed

## Risk

- [ ] No policy bypass path introduced
- [ ] Run/daemon parity preserved
- [ ] No new legacy compatibility path added
