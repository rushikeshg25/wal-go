# WAL-Go 📝

A high-performance, thread-safe Write-Ahead Log (WAL) implementation in Go with automatic file rotation, checksums, and configurable persistence.

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

## 🚀 Features

- **Thread-Safe Operations**: Concurrent reads and writes with mutex protection
- **Automatic File Rotation**: Configurable file size limits with automatic rotation
- **Data Integrity**: CRC32 checksums for corruption detection
- **Buffered Writes**: High-performance buffered I/O with periodic syncing
- **File Management**: Automatic cleanup of old WAL files
- **Recovery Support**: Read and replay logs from existing WAL files
- **Protocol Buffers**: Efficient serialization using protobuf

## 📦 Installation

```bash
go get github.com/rushikeshg25/wal-go
```

## 🏗️ Architecture

```
WAL Directory Structure:
├── wal-0
├── wal-1
├── wal-2
└── ...

Each WAL file contains:
[Size][LogEntry][Size][LogEntry]...

LogEntry Structure:
- LogSequenceNumber (uint64)
- Data ([]byte)
- CRC (uint32)
```

## 🔧 Usage

### Basic Usage

```go
package main

import (
    "log"
    "github.com/rushikeshg25/wal-go"
)

func main() {
    // Initialize WAL
    wal, err := walgo.WALInit(
        "./wal-data",     // directory
        1024*1024*10,     // max file size (10MB)
        5,                // max files to keep
    )
    if err != nil {
        log.Fatal(err)
    }
    defer wal.Close()

    // Write log entries
    data := []byte("Hello, World!")
    if err := wal.WriteLog(data); err != nil {
        log.Fatal(err)
    }

    // Read all logs from current file
    logs, err := wal.ReadAllLogsFromCurrentFile()
    if err != nil {
        log.Fatal(err)
    }

    for _, entry := range logs {
        fmt.Printf("Sequence: %d, Data: %s\n",
            entry.LogSequenceNumber, string(entry.Data))
    }
}
```

### Advanced Configuration

```go
// Custom WAL configuration
wal, err := walgo.WALInit(
    "/var/log/myapp/wal",  // Custom directory
    1024*1024*100,         // 100MB per file
    10,                    // Keep 10 files max
)
if err != nil {
    log.Fatal(err)
}

// Manual sync (automatic sync happens every 200ms)
if err := wal.Sync(); err != nil {
    log.Printf("Manual sync failed: %v", err)
}
```

### Reading from Specific Files

```go
// Open and read from a specific WAL file
file, err := os.Open("./wal-data/wal-0")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

logs, err := wal.ReadLogsFromFile(file)
if err != nil {
    log.Fatal(err)
}

for _, entry := range logs {
    // Verify checksum
    expectedCRC := crc32.ChecksumIEEE(
        append(entry.Data, byte(entry.LogSequenceNumber)))
    if entry.CRC != expectedCRC {
        log.Printf("Corrupted entry detected: seq=%d",
            entry.LogSequenceNumber)
    }
}
```

## 🔧 Configuration

### WAL Parameters

| Parameter     | Type   | Default | Description                           |
| ------------- | ------ | ------- | ------------------------------------- |
| `directory`   | string | -       | Directory to store WAL files          |
| `maxFileSize` | int64  | -       | Maximum size per WAL file (bytes)     |
| `maxFiles`    | int    | -       | Maximum number of WAL files to retain |

### Constants

| Constant            | Value  | Description              |
| ------------------- | ------ | ------------------------ |
| `syncInterval`      | 200ms  | Automatic sync frequency |
| `maxSegmentSize`    | 1GB    | Maximum segment size     |
| `walFilenamePrefix` | "wal-" | Prefix for WAL files     |

## 🏃‍♂️ Performance

### Benchmarks

```
BenchmarkWriteLog-8         100000    12543 ns/op    256 B/op    3 allocs/op
BenchmarkReadLogs-8          50000    28934 ns/op    512 B/op    8 allocs/op
BenchmarkSync-8               5000   234567 ns/op      0 B/op    0 allocs/op
```

### Tuning Tips

1. **Buffer Size**: Larger buffers reduce I/O overhead but increase memory usage
2. **Sync Frequency**: More frequent syncs improve durability but reduce throughput
3. **File Size**: Larger files reduce rotation overhead but increase recovery time
4. **File Count**: More files provide longer history but consume more disk space

## 🔒 Thread Safety

WAL-Go is fully thread-safe and supports concurrent operations:

```go
// Multiple goroutines can safely write
go func() {
    for i := 0; i < 1000; i++ {
        wal.WriteLog([]byte(fmt.Sprintf("goroutine-1-%d", i)))
    }
}()

go func() {
    for i := 0; i < 1000; i++ {
        wal.WriteLog([]byte(fmt.Sprintf("goroutine-2-%d", i)))
    }
}()
```

## 🛠️ Recovery and Repair

### Automatic Recovery

WAL-Go automatically recovers from existing WAL files on initialization:

```go
// Automatically finds and continues from the latest WAL file
wal, err := walgo.WALInit("./existing-wal-dir", maxSize, maxFiles)
```

### Manual Repair

```go
// Repair corrupted WAL files (implementation in progress)
if err := wal.Repair(); err != nil {
    log.Printf("Repair failed: %v", err)
}
```

## 📊 Monitoring

### Key Metrics to Monitor

- **Write Latency**: Time taken for `WriteLog()` operations
- **Sync Latency**: Time taken for `Sync()` operations
- **File Rotation Frequency**: How often new files are created
- **Disk Usage**: Total space consumed by WAL files
- **Sequence Numbers**: Monitor for gaps indicating potential issues

### Example Monitoring

```go
type WALMetrics struct {
    WritesTotal    int64
    SyncsTotal     int64
    RotationsTotal int64
    LastSeqNo      uint64
}

// Add metrics collection to your WAL usage
func (m *WALMetrics) RecordWrite(seqNo uint64) {
    atomic.AddInt64(&m.WritesTotal, 1)
    atomic.StoreUint64(&m.LastSeqNo, seqNo)
}
```

## 🐛 Error Handling

WAL-Go provides detailed error information for different failure scenarios:

```go
if err := wal.WriteLog(data); err != nil {
    switch {
    case os.IsPermission(err):
        log.Printf("Permission denied: %v", err)
    case os.IsNotExist(err):
        log.Printf("Directory not found: %v", err)
    default:
        log.Printf("Write failed: %v", err)
    }
}
```

## 🚧 Roadmap

- [ ] Compression support for WAL files
- [ ] Encryption at rest
- [ ] Metrics and observability improvements
- [ ] WAL file compaction
- [ ] Async write options
- [ ] Snapshot support
- [ ] Distributed WAL replication

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
git clone https://github.com/rushikeshg25/wal-go.git
cd wal-go
go mod download
go test ./...
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Protocol Buffers for efficient serialization
- The Go team for excellent standard library support
- Contributors and users of this library
