# Contributing

```bash
git clone https://github.com/praxicraft-platform/praxicraft-go.git
cd praxicraft-go
go test ./...
```

Guidelines:

- Thin wrapper around the [Assess Public API](https://docs.praxicraft.com).
- Keep HTTP mocked in tests — no live production calls in CI.
- Package name is `praxicraft`; module path is `github.com/praxicraft-platform/praxicraft-go`.
- Release notes: [RELEASING.md](RELEASING.md).
