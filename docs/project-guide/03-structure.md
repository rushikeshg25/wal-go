# Structure

## What lives where

The tracked baseline has 16 files. The root package contains the complete handwritten implementation and behavioral tests; `pb/` contains generated message code, and `tests/` contains only an empty package declaration. Supporting root documents record usage, acceptance and delivery history. There are no HTTP handlers, executable entry points, database adapters or CI/deployment files in this inventory.

```text
wal-go/
├── wal.go                 handle, segment management and public API
├── format.go              checksum, positioned writes and recovery parser
├── util.go                compatibility codecs
├── wal.proto              protobuf record schema
├── wal_test.go            four behavioral tests
├── pb/wal.pb.go            generated protobuf implementation
├── tests/wal_tests.go      empty package declaration
├── go.mod / go.sum         module and dependencies
├── Makefile                root-package test target
├── generate_pb.sh          schema generation command
├── README.md               public usage and limitations
├── V1.md                   acceptance contract
├── HISTORY.md              delivery chronology
└── docs/project-guide/     this six-file guide
```

## Root package

| File | Responsibility | Key exports | Called by |
| --- | --- | --- | --- |
| [wal.go](../../wal.go#L18) | Own handle state; scan directory; initialize, rotate, retain, synchronize, read and close. | `WAL`, `WALInit`, `ErrClosed`, `ErrPoisoned`; `WriteLog`, `WriteWALEntryToBuffer`, `Sync`, `Close`, `ReadAll`, `ReadAllLogsFromCurrentFile`, `ReadLogsFromFile`, `Repair`; legacy `InitExisingWAL` | Embedding applications, ticker, root tests |
| [format.go](../../format.go#L15) | Define v1 magic/record bound, CRC, full positioned writes and validated parser with optional partial-tail truncation. | No exports; `checksum`, `writeFull`, `scan` | `wal.go`; `util.go` uses checksum |
| [util.go](../../util.go#L8) | Legacy panic-on-error codecs and an internal checksum comparison helper. | `Marshal`, `UnMarshall` | External compatibility callers; internal `verifyCRC` currently has no caller |
| [wal.proto](../../wal.proto#L1) | Define the proto3 record and generated package location. | Schema message `WAL_Entry` | `protoc`, then append/recovery via generated Go |
| [wal_test.go](../../wal_test.go#L11) | Exercise rotation/retention/restart, incomplete tail and corruption, concurrent writes and close, and empty-segment sequence preservation. | `TestRotationRetentionRestart`, `TestTailAndCorruption`, `TestConcurrentWritesClose`, `TestEmptySegmentPreservesSequence` | `go test`, `make test` |
| [go.mod](../../go.mod#L1) | Module path, Go version and dependency requirements. | `github.com/rushikeshg25/wal-go` module | Go toolchain |
| [Makefile](../../Makefile#L1) | Run `go test .`. Declares `run` phony but provides no run recipe. | `test` target | Developer's `make test` |
| [generate_pb.sh](../../generate_pb.sh#L1) | Invoke `protoc --go_out=. wal.proto` from repository root. | None | Developer regenerating schema |
| [README.md](../../README.md#L1) | Library example, durability/format/ownership contracts, test command and limitations. | None | Library users |
| [V1.md](../../V1.md#L1) | Define v1 delivery scope and acceptance expectations. | None | Maintainers and reviewers |
| [HISTORY.md](../../HISTORY.md#L1) | Record baseline and v1 delivery commits with evidence links. | None | Maintainers reconstructing history |

## pb folder

[The pb directory](../../pb/) contains [wal.pb.go](../../pb/wal.pb.go#L1), generated from [wal.proto](../../wal.proto#L1). Generated internals are excluded from the handwritten-source inventory; the public `pb.WAL_Entry` fields are used by append, reads and compatibility helpers. The generated header records compiler versions; see [tooling](04-tech-stack.md#tooling).

## Tests folder

| File | Responsibility | Key exports | Called by |
| --- | --- | --- | --- |
| [tests/wal_tests.go](../../tests/wal_tests.go#L1) | Only `package tests`; no executable tests or helpers. It does not use the `_test.go` suffix. | None | Compiled by `go test ./...` as an empty package |

Actual behavioral coverage is in [the root tests](../../wal_test.go#L11). Those tests use package `walgo`, allowing direct access to private state and file helpers; the empty-segment test creates a segment under the private mutex ([wal_test.go:94](../../wal_test.go#L94)). There are no dedicated committed tests for old-format rejection, invalid initialization limits, injected storage failures, the public `Repair` method, or the compatibility APIs in the four tests present.

## Guide folder

| File | Responsibility | Key exports | Called by |
| --- | --- | --- | --- |
| [README.md](README.md) | Entry point, source tour, verification and open questions. | None | New maintainers |
| [01-architecture.md](01-architecture.md) | Components, contracts, persistence and scaling. | None | Guide readers |
| [02-flow.md](02-flow.md) | Startup through append/read/recovery/shutdown. | None | Guide readers |
| [03-structure.md](03-structure.md) | Complete significant-file inventory. | None | Guide readers |
| [04-tech-stack.md](04-tech-stack.md) | Manifest-backed stack and tooling. | None | Guide readers |
| [05-decisions.md](05-decisions.md) | Evidenced choices, inferences and gotchas. | None | Guide readers |

## Excluded

The lockfile [go.sum](../../go.sum), routine [.gitignore](../../.gitignore), and tracked [.DS_Store](../../.DS_Store) are omitted from responsibility tables. Generated protobuf internals are summarized above rather than treated as design-authored code. Working-tree material outside the tracked baseline and this guide is not part of this inventory.
