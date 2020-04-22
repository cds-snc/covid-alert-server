package retrieval

import (
	"context"
	"encoding/binary"
	"io"
	"time"

	pb "CovidShield/pkg/proto/covidshield"

	"google.golang.org/protobuf/proto"

	"github.com/Shopify/goose/logger"
)

var log = logger.New("retrieval")

const (
	// works out to 30 per key plus a tiny bit of overhead for the File / Header
	keyMessageSize = 30
	// caps out at 56 so this adds a small safety margin
	fileAndHeaderMessageSize = 100
	maxMessageSize           = 500 * 1024
	maxKeysPerFile           = (maxMessageSize - fileAndHeaderMessageSize) / keyMessageSize
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func SerializeTo(ctx context.Context, w io.Writer, keysByRegion map[string][]*pb.Key, startTimestamp, endTimestamp time.Time) error {
	files := []*pb.File{}

	start := startTimestamp.Unix()
	end := endTimestamp.Unix()

	batchNum := int32(1)
	for region, keys := range keysByRegion {
		for offset := 0; offset < len(keys); offset += maxKeysPerFile {
			thisBatchNum := batchNum
			thisRegion := region
			files = append(files, &pb.File{
				Header: &pb.Header{
					StartTimestamp: &start,
					EndTimestamp:   &end,
					Region:         &thisRegion,
					BatchNum:       &thisBatchNum,
				},
				Key: keys[offset:min(offset+maxKeysPerFile, len(keys))],
			})
			batchNum++
		}
	}

	for _, file := range files {
		l := int32(len(files))
		file.Header.BatchSize = &l
	}

	if len(files) == 0 {
		// We write out [0 0 0 0] if there were no files. This is a special case.
		return binary.Write(w, binary.BigEndian, uint32(0))
	}

	for _, file := range files {
		data, err := proto.Marshal(file)
		if err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}

	return nil
}
