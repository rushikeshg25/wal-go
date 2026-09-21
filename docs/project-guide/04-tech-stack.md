# Tech Stack

## Languages and runtimes

| Language or runtime | Version | Pinned at |
| --- | --- | --- |
| Go | `1.23.2` module language/toolchain requirement; no separate `toolchain` directive | [go.mod:3](../../go.mod#L3) |
| Protocol Buffers schema | `proto3` | [wal.proto:1](../../wal.proto#L1) |
| Bash | No version pin; schema-generation script shebang | [generate_pb.sh:1](../../generate_pb.sh#L1) |

## Frameworks and major libraries

| Library | Version | Used for | Used in |
| --- | --- | --- | --- |
| `google.golang.org/protobuf` | `v1.36.5`, [go.mod:5](../../go.mod#L5) | Marshal appended entries and unmarshal validated frames; generated message runtime | [wal.go:175](../../wal.go#L175), [format.go:92](../../format.go#L92), [pb/wal.pb.go](../../pb/wal.pb.go#L9) |
| `github.com/stretchr/testify` | `v1.10.0`, marked indirect in [go.mod:7](../../go.mod#L7) | Declared dependency, but no use in the current handwritten source/tests | [wal_test.go imports](../../wal_test.go#L3) use standard `testing` instead |
| Go standard library | Follows Go toolchain above | OS files, CRC32 IEEE, binary framing, mutex/channels/ticker, error joining, sorting | [wal.go imports](../../wal.go#L3), [format.go imports](../../format.go#L3) |

There is no application framework: exported Go methods are the integration surface ([wal.go](../../wal.go#L85)).

## Data and infrastructure

| Service | Role | Configured at |
| --- | --- | --- |
| Local filesystem | Numbered v1 segment files are the persistent store; no external datastore is contacted. | Directory, byte limit and file-count limit supplied to [WALInit](../../wal.go#L85) |
| File and directory fsync | Establish durability for file contents and segment-name creation/deletion. | [create](../../wal.go#L62), [retain](../../wal.go#L147), [Sync](../../wal.go#L221) |
| In-process ticker | Attempt periodic sync every 200 ms; no external scheduler. | [wal.go:229](../../wal.go#L229) |

No environment-variable loader, container or infrastructure manifest is present in [the inventory](03-structure.md). Filesystem durability semantics and exclusive ownership are deployment requirements, not managed by a service dependency.

## Tooling

| Tool | Role | Configured at |
| --- | --- | --- |
| Go test and race detector | Run tests across packages with `go test -race ./...`; version follows the Go toolchain requirement. | [README.md:24](../../README.md#L24), [go.mod:3](../../go.mod#L3), [wal_test.go](../../wal_test.go#L11) |
| Make | Convenience `test` target runs only `go test .`, without `-race`; Make version is not pinned. | [Makefile:1](../../Makefile#L1) |
| `protoc` and `protoc-gen-go` | Generate the checked-in Go message file; invocation does not install or pin either tool. | [generate_pb.sh:3](../../generate_pb.sh#L3) |
| Generated-code provenance | Header records `protoc v5.29.3` and `protoc-gen-go v1.36.6`; these are observed generator versions, not reproducible install pins. | [pb/wal.pb.go:1](../../pb/wal.pb.go#L1) |

## Notes

- Generated-code provenance (`protoc-gen-go v1.36.6`) differs from the runtime requirement (`v1.36.5`). The checked-in output includes compatibility checks and current tests compile; do not assume the generation script pins the original generator ([generated header and checks](../../pb/wal.pb.go#L1), [go.mod](../../go.mod#L5)).
- The current tests use real temporary directories and OS files, with no mocked filesystem or fault injection ([wal_test.go:11](../../wal_test.go#L11)). Their passing result does not establish behavior under every disk or crash failure.
- `make run` has no runnable application recipe despite its phony declaration ([Makefile:5](../../Makefile#L5)); embed the library in an application.
