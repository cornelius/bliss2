# Contributing

## Getting started

**Prerequisites:** Go 1.21+

```sh
git clone https://github.com/cornelius/bliss2
cd bliss2
make build
make test
```

## Before submitting

Run both before every commit:

```sh
make test
make build
```

## Documentation

Read these before making changes:

- **CLI.md** — command specification and output design. The e2e tests verify this contract.
- **SPEC.md** — file format and store structure.
- **ARCHITECTURE.md** — package layout, dependencies, testing strategy.

## Tests

New behavior goes in `e2e/` (invokes the real binary, verifies the user contract as described in CLI.md). Internal logic goes in `*_test.go` alongside the package. See ARCHITECTURE.md § Testing for the full strategy.

**Never test manually.** If something needs verifying, write a test.

## Code conventions

- Nothing outside `internal/store` constructs paths into `~/.bliss2/` or touches store files directly.
- Return clean user-facing error messages. No stack traces shown to users.
- Comments only where the code isn't self-explanatory — focus on why, not what.
- Keep dependencies minimal. Any addition needs a clear reason.
