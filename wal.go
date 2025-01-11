package walgo

import (
	"bufio"
	"context"
	"os"
	"sync"
	"time"
)

type WAL struct{
	dir string
	currentSegment *os.File
	lock sync.Mutex
	bufWriter *bufio.Writer
	ctx context.Context
	cancel context.CancelFunc
	macSegments int

}

const (
	syncInterval = time.Millisecond * 500
	segementPrefix = "wal-"
)