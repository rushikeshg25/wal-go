# wal-go Project Guide

> Generated: 2026-09-21 from commit `92931fd`.

## What this is

wal-go is a Go library for writing checksummed byte records to size-limited segments on a local filesystem ([public API](../../wal.go#L85)). An embedding application owns the directory, chooses retention limits, and explicitly synchronizes records when it needs durability ([append and sync](../../wal.go#L165)). The v1 format rejects unversioned files and repairs only incomplete record tails in the newest segment ([parser](../../format.go#L41)).

## Run it

Use Go 1.23.2 or a compatible newer toolchain, as declared in [go.mod](../../go.mod#L3). This is a library; there is no executable or environment-variable configuration.

```bash
go mod download
go test -race ./...
go build ./...
```

An application imports `github.com/rushikeshg25/wal-go`, conventionally as `walgo`, then follows the [README example](../../README.md#L3): `WALInit`, `WriteLog`, `Sync`, `ReadAll`, `Close`. Check every returned error, including `Close`. Initialization takes a directory, a segment size of at least 64 bytes, and a retained-file count of at least one ([validation](../../wal.go#L85)). The directory must be exclusively owned by this application; existing unversioned data needs a separate migration plan ([compatibility contract](../../README.md#L16)).

Verification for this guide: `GOCACHE=/private/tmp/go-v1-suite/go-cache go test -race ./...` passed (cached result). A separate temporary executable called `WALInit` on three-byte and 32-byte unversioned `wal-0` files: both were rejected, and byte-for-byte readback confirmed neither changed. This probe was not added to the repository. It exercised the [header checks](../../format.go#L48); it is not a crash or power-loss durability test.

## The five-file tour

| # | File | Why this one | Then look at |
| --- | --- | --- | --- |
| 1 | [wal.go](../../wal.go#L85) | Start with initialization, then follow the exported append/read/lifecycle methods. | [Startup](02-flow.md#startup) |
| 2 | [wal.proto](../../wal.proto#L4) | Learn the sequence, payload and checksum fields. | [Data model](01-architecture.md#data-model) |
| 3 | [format.go](../../format.go#L41) | Trace byte framing, validation and narrow tail recovery. | [Reading and repair](02-flow.md#reading-and-repair) |
| 4 | [util.go](../../util.go#L8) | Recognize legacy panic-based codecs that the recovery path avoids. | [Gotchas](05-decisions.md#gotchas) |
| 5 | [wal_test.go](../../wal_test.go#L11) | Read the executable examples of retention, corruption and concurrency. | [Tests folder and test coverage](03-structure.md#tests-folder) |

## Reading order for this guide

1. [Architecture](01-architecture.md) — ownership, contracts and persistent state.
2. [Flow](02-flow.md) — initialization, append, rotation, reads and shutdown.
3. [Structure](03-structure.md) — source and supporting-file inventory.
4. [Tech stack](04-tech-stack.md) — version pins and build tools.
5. [Decisions](05-decisions.md) — evidence, inferred rationale and traps.

## Open questions

- What application checkpoint makes deletion of old segments safe? [Retention](../../wal.go#L147) only uses the file-count limit; no consumer acknowledgement is modeled.
- How will old unversioned records be migrated? [The parser](../../format.go#L56) rejects them, and [the README](../../README.md#L16) supplies no migration utility.
- What filesystem/platform guarantees and failure-injection coverage are required before production use? [Creation and fsync](../../wal.go#L62) depend on OS behavior; [current tests](../../wal_test.go#L11) do not inject short writes, failed fsync, or process/power loss.
- Should lower-level `WriteWALEntryToBuffer` enforce the encoded-size cap for protobuf unknown fields? [Append](../../wal.go#L169) bounds `Data` and segment capacity, while [recovery](../../format.go#L72) rejects encoded records over 64 MiB. Ordinary `WriteLog` constructs only known fields.
