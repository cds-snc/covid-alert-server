package retrieval

import (
	"archive/zip"
	"context"
	"io"
	"time"

	pb "github.com/CovidShield/server/pkg/proto/covidshield"

	"github.com/Shopify/goose/logger"
	"google.golang.org/protobuf/proto"
)

var log = logger.New("retrieval")

const (
	maxKeysPerFile = 750000
)

var (
	signatureAlgorithm     = "1.2.840.10045.4.3.2" // required by protocol
	verificationKeyVersion = "v1"
	verificationKeyID      = "302"
	binHeader              = []byte("EK Export v1    ")
	binHeaderLength        = 16
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// It's still really unclear to me when A/G wants us to use MCC and when we're
// expected/permitted to use some other identifier. It would be great to get
// more clarity on this.
func transformRegion(reg string) string {
	if reg == "302" {
		return "CA"
	}
	return reg
}

func SerializeTo(
	ctx context.Context, w io.Writer,
	keys []*pb.TemporaryExposureKey,
	region string,
	startTimestamp, endTimestamp time.Time,
	signer Signer,
) (int, error) {
	zipw := zip.NewWriter(w)

	one := int32(1)

	start := uint64(startTimestamp.Unix())
	end := uint64(endTimestamp.Unix())

	sigInfo := &pb.SignatureInfo{
		VerificationKeyVersion: &verificationKeyVersion,
		VerificationKeyId:      &verificationKeyID,
		SignatureAlgorithm:     &signatureAlgorithm,
	}

	region = transformRegion(region)

	tekExport := &pb.TemporaryExposureKeyExport{
		StartTimestamp: &start,
		EndTimestamp:   &end,
		Region:         &region,
		BatchNum:       &one,
		BatchSize:      &one,
		SignatureInfos: []*pb.SignatureInfo{sigInfo},
		Keys:           keys,
	}

	exportBinData, err := proto.Marshal(tekExport)
	if err != nil {
		return -1, err
	}

	sig, err := signer.Sign(exportBinData)
	if err != nil {
		return -1, err
	}

	sigList := &pb.TEKSignatureList{
		Signatures: []*pb.TEKSignature{&pb.TEKSignature{
			SignatureInfo: sigInfo,
			BatchNum:      &one,
			BatchSize:     &one,
			Signature:     sig,
		}},
	}
	exportSigData, err := proto.Marshal(sigList)
	if err != nil {
		return -1, err
	}

	totalN := 0

	f, err := zipw.Create("export.bin")
	if err != nil {
		return -1, err
	}
	n, err := f.Write(binHeader)
	if err != nil {
		return -1, err
	}
	totalN += n
	if n != binHeaderLength {
		panic("header len")
	}
	n, err = f.Write(exportBinData)
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
