package walgo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func createSegmentFile(dir string, segmentId int)(*os.File,error){
	filePath:=filepath.Join(dir,fmt.Sprintf("segment-%d",segmentId))
	file,err:=os.Create(filePath)
	if err!=nil{
		return nil,err
	}
	return file,nil
}

func findLastSegmentIndexinFiles(files []string)(int,error){
	var lastId int
	for _,file:=range files{
		_,fileName:=filepath.Split(file)
		segmentId,err:=strconv.Atoi(strings.TrimPrefix(fileName,segmentFilenamePrefix))
		if err!=nil{
			return 0,err
		}
		if segmentId>lastId{
			lastId=segmentId
		}
	}
	return lastId,nil
}