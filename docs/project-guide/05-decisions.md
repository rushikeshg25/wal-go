# Decisions

These choices are grounded in current code. Explicit comments/contracts are distinguished from inferred rationale; code establishes behavior but does not by itself prove the author's motivation.

## Reject old formats at the header boundary

- **What:** Require the exact eight-byte v1 magic/version/header-size prefix and a full 16-byte header.
- **Evidence:** [format.go:48](../../format.go#L48) returns errors before parsing frames; [README.md:16](../../README.md#L16) explicitly rejects unversioned files and provides no automatic migration tool.
- **Why:** The README states that rejection avoids silently interpreting incompatible records.
- **Tradeoff:** The parser has one format to validate, but existing users need a fresh directory or an explicit migration solution. A short old file is reported as a truncated header; a longer nonmatching file is reported as unsupported format.
- **Confidence:** Confirmed by code and documentation; temporary three-byte and 32-byte file probes also verified rejection without modification.

## Carry the prior sequence in every segment header

- **What:** Write the current last sequence into each new segment header, and recover its value even when the segment has no entries.
- **Evidence:** [wal.go:67](../../wal.go#L67), [format.go:60](../../format.go#L60), and [TestEmptySegmentPreservesSequence](../../wal_test.go#L94).
- **Why, apparently:** Preserve sequence continuity after rotation/retention leaves an empty newest segment; the named test directly demonstrates this intent.
- **Tradeoff:** No separate sequence manifest is needed. The base field itself has no separate checksum; after older history is deleted, an empty sole segment has no record checksum against which to validate that base.
- **Confidence:** Confirmed behavior and test; the no-manifest benefit is inferred.

## Serialize every operation with one mutex

- **What:** Append, reads, explicit repair, sync and close share the handle lock; the ticker calls the same public `Sync` method.
- **Evidence:** [wal.go:207](../../wal.go#L207), [wal.go:221](../../wal.go#L221), [wal.go:242](../../wal.go#L242), [wal.go:256](../../wal.go#L256).
- **Why, apparently:** Inferred: a single critical section keeps file rotation, offsets, sequence allocation and lifecycle state consistent.
- **Tradeoff:** Simple per-handle concurrency and race-tested writes; a slow full-log read or fsync blocks other operations. This is not an OS lock and does not protect multiple handles/processes sharing a directory.
- **Confidence:** Implementation confirmed; rationale inferred. [Concurrent-write test](../../wal_test.go#L69) covers one handle.

## Separate accepted writes from durable writes

- **What:** Append writes immediately through `WriteAt`; explicit sync, background sync and close call fsync.
- **Evidence:** [wal.go:197](../../wal.go#L197), [wal.go:221](../../wal.go#L221), [README.md:12](../../README.md#L12).
- **Why, apparently:** Inferred: avoid a required fsync per record while allowing applications to request durability at their own boundary.
- **Tradeoff:** Higher batching potential, with a period during which accepted records may be lost on a crash. The ticker's 200 ms interval is an attempt cadence, not a deadline guarantee.
- **Confidence:** Contract confirmed; performance motivation inferred.

## Poison ambiguous I/O state and require reopen

- **What:** Record append/rotation/sync failures, reject subsequent operations with `ErrPoisoned`, and still attempt final sync/close. Input-validation errors return without poisoning.
- **Evidence:** [check/fail](../../wal.go#L132), [append error paths](../../wal.go#L169), [Close](../../wal.go#L242); [Repair](../../wal.go#L307) additionally poisons on scan failure.
- **Why, apparently:** Inferred: prevent further use of offsets or durability assumptions after an I/O result makes them uncertain.
- **Tradeoff:** Callers need a close/reopen recovery path; repair cannot be invoked through an already poisoned handle. Read errors alone do not set poison. Background errors are only indirectly visible on a later call.
- **Confidence:** Behavior confirmed; rationale inferred. Failure injection is not covered by the four [current tests](../../wal_test.go#L11).

## Repair only incomplete final frames

- **What:** Truncate a partial length prefix or short body only at the current/newest segment tail. Reject oversized length, invalid protobuf, wrong checksum/sequence and malformed header.
- **Evidence:** [format.go:63](../../format.go#L63), [startup repair flag](../../wal.go#L102), [Repair comment](../../wal.go#L300), [tail/corruption test](../../wal_test.go#L43).
- **Why, apparently:** Inferred: a partial final frame is compatible with interrupted append; complete bad data has insufficient evidence for safe automatic deletion.
- **Tradeoff:** Narrow automatic recovery preserves complete-corruption evidence, but operators cannot use `Repair` as a general salvage tool. A torn header is unrecoverable through this parser.
- **Confidence:** Scope confirmed by comment/code; rationale inferred.

## Sync new segment metadata before deleting history

- **What:** On rotation, sync the old file, create and sync the new header and WAL directory, close the old file, then delete excess segments and sync the directory again.
- **Evidence:** [wal.go:182](../../wal.go#L182), [create](../../wal.go#L62), [retain](../../wal.go#L147).
- **Why, apparently:** Inferred: establish a persisted next-segment header before removing older sequence history.
- **Tradeoff:** Rotation incurs several synchronous operations. Retention is based only on count, so it may discard application-needed recovery history and runs before the new record is appended. The parent of a newly created WAL directory is not fsynced by the library.
- **Confidence:** Ordering confirmed; rationale inferred. Ownership/checkpoint limitation is explicit in [README.md:20](../../README.md#L20).

## Keep source-compatible names while changing behavior

- **What:** Keep `WriteWALEntryToBuffer`, `InitExisingWAL`, `Marshal` and `UnMarshall` exports. The first appends directly; `InitExisingWAL` ignores its supplied `files` argument and simply discovers the highest numbered segment.
- **Evidence:** [wal.go:213](../../wal.go#L213), [wal.go:315](../../wal.go#L315), [util.go:8](../../util.go#L8), [README.md:22](../../README.md#L22).
- **Why:** The README explicitly calls out source compatibility for the old codec helpers; preserving the other names for the same reason is inferred.
- **Tradeoff:** Existing names remain callable but can mislead: there is no user-facing memory buffer, and the old initialization helper does not open or validate a WAL.
- **Confidence:** Behavior confirmed; compatibility rationale partly documented, partly inferred.

## Gotchas

- **Read and recovery are different validation scopes:** `WALInit` checks inter-segment numbering/base continuity; `ReadAll` only parses files individually. Single-file methods can return a validated prefix together with an error; callers must handle the error ([startup](../../wal.go#L107), [reads](../../wal.go#L256)).
- **Repeated close loses the first error:** only the first `Close` returns joined poison/sync/close errors; subsequent closes wait and return nil ([wal.go:244](../../wal.go#L244)).
- **Payload bound differs from encoded-frame bound:** append checks `len(Data) <= 64 MiB - 64` and total segment capacity, while scan caps encoded length at 64 MiB. A supplied protobuf entry can also contain unknown fields; the lower-level append method does not explicitly compare marshaled size to `maxRecord` ([wal.go:169](../../wal.go#L169), [format.go:72](../../format.go#L72), [generated unknown fields](../../pb/wal.pb.go#L29)).
- **Retention can delete on open:** reopening with a lower `maxFiles` scans existing files and then removes old ones immediately ([wal.go:125](../../wal.go#L125)).
- **Legacy codecs can panic:** recovery uses error-returning `proto.Unmarshal` directly rather than exported `UnMarshall` ([util.go:16](../../util.go#L16), [format.go:93](../../format.go#L93)).
- **The test directory is a placeholder:** the four real tests live in the root file, and `make test` omits race detection ([tests/wal_tests.go](../../tests/wal_tests.go#L1), [wal_test.go](../../wal_test.go#L11), [Makefile](../../Makefile#L1)).

## Conventions

- New public handle methods should follow the current lock/check pattern before using mutable file state ([ReadAll](../../wal.go#L274)); close intentionally handles lifecycle state itself.
- Keep recovery errors explicit rather than using panic-based compatibility codecs ([scan](../../format.go#L92)).
- Match the current tests' real temporary-directory setup and public behavior assertions; access private helpers only when arranging otherwise difficult states, as in the empty-segment test ([wal_test.go:94](../../wal_test.go#L94)). These are recommendations inferred from the existing code, not a separate repository coding policy.
