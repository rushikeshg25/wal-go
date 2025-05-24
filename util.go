package walgo

import (
	"hash/crc32"

	"github.com/rushikeshg25/wal-go/pb"
	"google.golang.org/protobuf/proto"
)

func Marshal(logEntry *pb.WAL_Entry) []byte {
	marsheledLogEntry, err := proto.Marshal(logEntry)
	if err != nil {
		panic("Marshalling failed")
	}
	return marsheledLogEntry
}

func UnMarshall(data []byte) *pb.WAL_Entry {
	logEntry := &pb.WAL_Entry{}
	err := proto.Unmarshal(data, logEntry)
	if err != nil {
		panic("Unmarshalling failed")
	}
	return logEntry
}

func verifyCRC(entry *pb.WAL_Entry) bool {
	actualCRC := crc32.ChecksumIEEE(append(entry.GetData(), byte(entry.GetLogSequenceNumber())))

	return entry.CRC == actualCRC
}
