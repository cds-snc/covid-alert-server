package retrieval

import (
	"archive/zip"
	"context"
	"io"
	"time"

	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"
	"google.golang.org/protobuf/proto"
)

func SerializeOutbreakEventsTo(
	ctx context.Context, w io.Writer,
	locations []*pb.OutbreakEvent,
	startTimestamp, endTimestamp time.Time,
	signer Signer,
) (int, error) {
	zipw := zip.NewWriter(w)

	start := uint64(startTimestamp.Unix())
	end := uint64(endTimestamp.Unix())

	outbreakEventExport := &pb.OutbreakEventExport{
		StartTimestamp: &start,
		EndTimestamp:   &end,
		Locations:      locations,
	}

	exportBinData, err := proto.Marshal(outbreakEventExport)
	if err != nil {
		return -1, err
	}

	sig, err := signer.Sign(exportBinData)
	if err != nil {
		return -1, err
	}

	sigExport := &pb.OutbreakEventExportSignature{
		Signature: sig,
	}
	exportSigData, err := proto.Marshal(sigExport)
	if err != nil {
		return -1, err
	}

	totalN := 0

	f, err := zipw.Create("export.bin")
	if err != nil {
		return -1, err
	}
	n, err := f.Write(exportBinData)
	if err != nil {
		return -1, err
	}
	totalN += n
	if n != len(exportBinData) {
		panic("len")
	}
	f, err = zipw.Create("export.sig")
	if err != nil {
		return -1, err
	}
	n, err = f.Write(exportSigData)
	if err != nil {
		return -1, err
	}
	totalN += n
	if n != len(exportSigData) {
		panic("len")
	}

	return totalN, zipw.Close()
}
