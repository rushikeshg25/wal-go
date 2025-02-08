package walgo

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type WAL struct {
	directory           string
	currentSegment      *os.File
	lock                sync.Mutex
	lastSequenceNo      uint64
	bufWriter           *bufio.Writer
	syncTimer           *time.Timer
	shouldFsync         bool
	maxFileSize         int64
	maxSegments         int
	currentSegmentIndex int
	ctx                 context.Context
	cancel              context.CancelFunc
}

const (
	segmentFilenamePrefix = "segment"
	syncInterval = 200*time.Millisecond
)


func OpenWAL(directory string, enableFsync bool, maxFileSize int64, maxSegments int) (*WAL, error){
	if err:=os.MkdirAll(directory,0755);err!=nil{
		return nil,err
	}

	files,err:=filepath.Glob(filepath.Join(directory,segmentFilenamePrefix+"*"))
	if err!=nil{
		return nil,err
	}

	lastSegmentId:=0
	if len(files)>0{
		lastSegmentId,err=findLastSegmentIndexinFiles(files)
		if err!=nil{
			return nil,err
		}
	}else{
		file,err:=createSegmentFile(directory,0)
		if err!=nil{
			return nil,err
		}

		if err:=file.Close();err!=nil{
			return nil,err
		}
	}

	filePath:=filepath.Join(directory,fmt.Sprintf("%s%d",segmentFilenamePrefix,lastSegmentId))
	file,err:=os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err!=nil{
		return nil,err
	}

	ctx,cancel:=context.WithCancel(context.Background())

	wal:=&WAL{
		directory: directory,
		currentSegment: file,
		lastSequenceNo: 0,
		bufWriter: bufio.NewWriter(file),
		syncTimer: time.NewTimer(syncInterval),
		shouldFsync: enableFsync,
		maxFileSize: maxFileSize,
		maxSegments: maxSegments,
		currentSegmentIndex: lastSegmentId,
		ctx: ctx,
		cancel: cancel,
	}
    
	return wal,nil
	
}

