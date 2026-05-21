Prefer make targets over direct commands. Run `make help` to see available targets.

Run `make can-release` before wrapping up substantial changes.

Keep `cmd/buns/main.go` thin; put CLI commands in `internal/cli`, execution flow in `internal/exec`, and cache/version logic in their existing packages.

Detroit-style tests: assert on observable behaviour and stub only at the boundary.
