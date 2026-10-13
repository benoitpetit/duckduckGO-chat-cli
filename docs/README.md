# Project documentation

Start with the guide that matches what you are doing:

- [User guide](USAGE.md): chat commands (including multi-model coordination), settings, voice, dashboard, API, and local data.
- [Build and release guide](BUILD.md): development setup, checks, and release builds.

## API reference

The local HTTP API specification is generated from annotations in `internal/api/docs.go`.
The generated files are `docs.go`, `swagger.json`, and `swagger.yaml` in this directory.

`POST /api/v1/chat` retains its JSON response. For incremental output, use `POST /api/v1/chat/stream`: it emits `start`, `delta`, then `done` or `error` server-sent events. Join `delta` values to rebuild the response; `done` carries model and processing metadata. The stream uses the same API-key and CORS middleware, and a disconnected client cancels generation.

When the API is running, open the Swagger UI at:

```text
http://127.0.0.1:8080/doc/index.html
```

To update the specification, edit the annotations in `internal/api/docs.go`, then run:

```sh
./scripts/generate-docs.sh
```

Commit the annotations and generated files together. The CI and release checks compare the generated output with its source.
