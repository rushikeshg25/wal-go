package walgo

import (
	"fmt"
	"os"
	"path/filepath"
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
	
}