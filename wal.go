package walgo

import (
	"bufio"
	"context"
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
		lastSegmentId,err:=findLastSegmentIndexinFiles(files)
		if err!=nil{
			return nil,err
		}
	}else{
		file,err:=createSegmentFile(directory,0)
		if err!=nil{
			return nil,err
		}

	}


	// return &WAL{
	// 	directory,

	// }
}

