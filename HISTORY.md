# Segmented write-ahead log history

> Scope: repository baseline through the v1 delivery branch.
> Last updated: 2026-09-21. Dates below retain Git author timezone offsets.

## At a glance

The v1 scope and acceptance contract is recorded in [V1.md](V1.md). Current behavior and limitations are documented in [README.md](README.md).

## Timeline

### 2025-01-09T16:48:55+05:30: Initial commit

- **What happened:** The repository records `Initial commit`.
- **Evidence:** [commit c7c0d4d575](https://github.com/rushikeshg25/wal-go/commit/c7c0d4d5758ce8a9cf82d4a17f46ac110b410259).
- **Confidence:** Confirmed by Git history; the title alone does not establish runtime correctness.

### 2026-09-21T13:23:54+05:30: docs: define wal-go v1 contract

- **What happened:** The repository records `docs: define wal-go v1 contract`.
- **Evidence:** [commit 0a3a1b0fc8](https://github.com/rushikeshg25/wal-go/commit/0a3a1b0fc8562fc46ecaaba22c3666cfa641b686).
- **Confidence:** Confirmed by Git history; the title alone does not establish runtime correctness.

### 2026-09-21T13:44:14+05:30: feat: define versioned bounded WAL records with full sequence checksums

- **What happened:** The repository records `feat: define versioned bounded WAL records with full sequence checksums`.
- **Evidence:** [commit 1a6bb55212](https://github.com/rushikeshg25/wal-go/commit/1a6bb5521215953668e5b824dc6b805e92bfba6d).
- **Confidence:** Confirmed by Git history; the title alone does not establish runtime correctness.

### 2026-09-21T13:44:14+05:30: feat: implement durable segmented append retention recovery and shutdown

- **What happened:** The repository records `feat: implement durable segmented append retention recovery and shutdown`.
- **Evidence:** [commit b0cf6d7e89](https://github.com/rushikeshg25/wal-go/commit/b0cf6d7e89f3853ef5a640dd450d3a5ba2209346).
- **Confidence:** Confirmed by Git history; the title alone does not establish runtime correctness.

### 2026-09-21T13:44:14+05:30: test: verify restart sequence retention corruption and concurrent WAL writes

- **What happened:** The repository records `test: verify restart sequence retention corruption and concurrent WAL writes`.
- **Evidence:** [commit da2b63d7ba](https://github.com/rushikeshg25/wal-go/commit/da2b63d7bac5961f4aa29658ba521c5a600155f8).
- **Confidence:** Confirmed by Git history; the title alone does not establish runtime correctness.

### 2026-09-21T14:04:58+05:30: docs: document wal-go v1 usage and limitations

- **What happened:** The repository records `docs: document wal-go v1 usage and limitations`.
- **Evidence:** [commit 68cb538087](https://github.com/rushikeshg25/wal-go/commit/68cb5380873f5795735cd6bd022cee3b03ebb226).
- **Confidence:** Confirmed by Git history; the title alone does not establish runtime correctness.

## Delivery verification

Passed `go test -race ./...`. [wal_test.go](wal_test.go) covers retained sequence continuity, empty-segment restart, tail recovery, checksum rejection, concurrent writes and close.

## Turning points

The v1 contract made failure behavior, lifecycle semantics and executable verification part of the delivery. The new tests and README describe the resulting boundaries.

## Open questions

No production deployment or long-running operational validation was performed as part of this delivery.
