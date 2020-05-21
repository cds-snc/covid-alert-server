package retrieval

import (
	"archive/zip"
	"context"
	"io"
	"strings"
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
	appBundleID            = "com.shopify.covid-shield"
	androidPackage         = "com.covidshield"
	signatureAlgorithm     = "ecdsa-with-SHA256" // required by protocol
	verificationKeyVersion = "v1"
	verificationKeyID      = "key-0"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func SerializeTo(
	ctx context.Context, w io.Writer,
	keysByRegion map[string][]*pb.TemporaryExposureKey,
	startTimestamp, endTimestamp time.Time,
	signer Signer,
) error {
	zipw := zip.NewWriter(w)

	one := int32(1)

	var regions []string
	for region, _ := range keysByRegion {
		regions = append(regions, region)
	}
	regionSpec := strings.Join(regions, "+")

	start := uint64(startTimestamp.Unix())
	end := uint64(endTimestamp.Unix())

	sigInfo := &pb.SignatureInfo{
		AppBundleId:            &appBundleID,
		AndroidPackage:         &androidPackage,
		VerificationKeyVersion: &verificationKeyVersion,
		VerificationKeyId:      &verificationKeyID,
		SignatureAlgorithm:     &signatureAlgorithm,
	}

	tekExport := &pb.TemporaryExposureKeyExport{
		StartTimestamp: &start,
		EndTimestamp:   &end,
		Region:         &regionSpec,
		BatchNum:       &one,
		BatchSize:      &one,
		SignatureInfos: []*pb.SignatureInfo{sigInfo},
	}

	// TODO: What on earth are we supposed to do if we have more than 750,000
	// keys? I *have* to assume G/A is going to revise this protocol again in the
	// next few days.
	for _, keys := range keysByRegion {
		tekExport.Keys = append(tekExport.Keys, keys...)
	}

	exportBinData, err := proto.Marshal(tekExport)
	if err != nil {
		return err
	}

	sig, err := signer.Sign(exportBinData)
	if err != nil {
		return err
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
		return err
	}

	f, err := zipw.Create("export.bin")
	if err != nil {
		return err
	}
	n, err := f.Write(exportBinData)
	if err != nil {
		return err
	}
	if n != len(exportBinData) {
		panic("len")
	}
	f, err = zipw.Create("export.sig")
	if err != nil {
		return err
	}
	n, err = f.Write(exportSigData)
	if err != nil {
		return err
	}
	if n != len(exportSigData) {
		panic("len")
	}

	return zipw.Close()
}
