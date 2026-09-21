# Segmented write-ahead log v1

```go
wal, err := walgo.WALInit("./wal-data", 10*1024*1024, 5)
if err != nil { panic(err) }
if err := wal.WriteLog([]byte("event")); err != nil { panic(err) }
if err := wal.Sync(); err != nil { panic(err) }
entries, err := wal.ReadAll()
if err := wal.Close(); err != nil { panic(err) }
```

Writes, reads, rotation, sync and close are serialized. `WriteLog` copies the payload through protobuf encoding and assigns increasing sequence numbers. Accepted writes are visible immediately but durable only after successful `Sync` or `Close`; a background ticker also calls fsync every 200ms. The first write/sync failure poisons the handle; reopen is required. Close stops the ticker, syncs and closes the file, reports errors and is idempotent.

Segments use a new explicit v1 format: 16-byte `WALG` header (version 1, header size 16), including the preceding uint64 sequence number, followed by little-endian uint32 lengths and protobuf entries. CRC32 covers all eight sequence bytes plus payload. Entries are bounded to 64 MiB including encoding; an entry must fit within a segment. `maxFileSize` must be at least 64 bytes and `maxFiles` at least one.

**Compatibility:** Existing unversioned WAL files are rejected. Use a new directory for v1; no automatic migration tool is supplied. This avoids silently interpreting incompatible records.

Startup validates every retained segment, checks sequence continuity and checksums, and truncates only an incomplete tail in the last segment. Complete corruption and damage in older segments fail open without repair. `Repair` performs that narrow tail repair on the current segment; it cannot recover arbitrary corruption. A segment header retains sequence continuity even when an empty newest segment is the only retained file.

Rotation syncs the previous file, creates/syncs the next header and syncs the directory before removing excess old segments. Retention deliberately deletes old history: the caller must choose limits consistent with its recovery needs. There is no database checkpoint coordination. One process must exclusively own the directory. Directory creation in its own parent is not fsynced by this library.

`ReadAll` returns retained entries in order. `ReadAllLogsFromCurrentFile` scans the active segment. `ReadLogsFromFile` validates a supplied v1 file without changing the file's read cursor. Legacy `WriteWALEntryToBuffer` now appends directly and requires the next sequence and matching CRC; use `WriteLog` for ordinary calls. The old panic-on-error codec helpers remain for source compatibility and are not used by the recovery parser.

Run `go test -race ./...`. Tests cover rotation, retention, restart continuity, incomplete tails, checksum corruption, concurrent writes and repeated close. V1 has no cross-process locking or transactional multi-record append.
