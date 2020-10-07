package persistence

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"io/ioutil"
	"math/big"
	"regexp"
	"strings"
	"time"

	pb "github.com/cds-snc/covid-alert-server/pkg/proto/covidshield"

	"github.com/Shopify/goose/logger"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/nacl/box"
)

// ErrTooManyKeys is returned when the client tries to insert one or more keys
// past their limit, assigned on keypair creation. The entire batch is rejected.
var ErrTooManyKeys = errors.New("key limit for keypair exceeded")

// Conn mediates all access to a MySQL/CloudSQL connection. It exposes a
// method for each query we support. The one exception is database
// creation/migrations, which are handled separately.
type Conn interface {
	// Return keys that were SUBMITTED to the Diagnosis Server during the specified
	// UTC date.
	//
	// Only returns keys that correspond to a Key for a date
	// less than 14 days ago.
	FetchKeysForHours(string, uint32, uint32, int32) ([]*pb.TemporaryExposureKey, error)
	StoreKeys(*[32]byte, []*pb.TemporaryExposureKey, context.Context) error
	NewKeyClaim(context.Context, string, string, string) (string, error)
	ClaimKey(string, []byte, context.Context) ([]byte, error)
	PrivForPub([]byte) ([]byte, error)

	CheckClaimKeyBan(string) (triesRemaining int, banDuration time.Duration, err error)
	ClaimKeySuccess(string) error
	ClaimKeyFailure(string) (triesRemaining int, banDuration time.Duration, err error)

	DeleteOldDiagnosisKeys() (int64, error)
	DeleteOldEncryptionKeys() (int64, error)
	DeleteOldFailedClaimKeyAttempts() (int64, error)

	CountClaimedOneTimeCodes() (int64, error)
	CountDiagnosisKeys() (int64, error)
	CountUnclaimedOneTimeCodes() (int64, error)

	CountUnclaimedEncryptionKeysByOriginator() ([]CountByOriginator, error)
	CountExhaustedEncryptionKeysByOriginator() ([]CountByOriginator, error)
	CountExpiredClaimedEncryptionKeysByOriginator() ([]CountByOriginator, error)

	SaveEvent(event Event) error
	GetServerEventsByType(eventType EventType, startDate string) ([]Events, error)

	Close() error
}

type conn struct {
	db *sql.DB
}

var log = logger.New("db")

const (
	maxConnLifetime = 5 * time.Minute
	maxOpenConns    = 100
	maxIdleConns    = 10
)

// Dial establishes a MySQL/CloudSQL connection and returns a Conn object,
// wrapping each available query.
func Dial(url string) (Conn, error) {
	if strings.Contains(url, "?") {
		url += "&parseTime=true"
	} else {
		url += "?parseTime=true"
	}

	// Check if we are connecting to RDS
	if strings.Contains(url, "rds.amazonaws.com") {

		rootCertPool := x509.NewCertPool()
		pem, err := ioutil.ReadFile("/etc/aws-certs/rds-ca-2019-root.pem")

		if err != nil {
			log(nil, err).Fatal("AWS RDS Cert bundle not found")
		}

		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			log(nil, err).Fatal("Could not append certs")
		}

		re := regexp.MustCompile(`tcp\((.*)\)`)
		match := re.FindStringSubmatch(url)

		if len(match) > 0 {
			mysql.RegisterTLSConfig("custom", &tls.Config{
				ServerName: match[1],
				RootCAs:    rootCertPool,
			})
			url += "&tls=custom"
		}
	}

	db, err := sql.Open("mysql", url)
	if err != nil {
		log(nil, err).Fatal("Could not connect to database")
	}
	db.SetConnMaxLifetime(maxConnLifetime)
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	return &conn{db: db}, nil
}

func (c *conn) DeleteOldDiagnosisKeys() (int64, error) {
	return deleteOldDiagnosisKeys(c.db)
}

func (c *conn) CountUnclaimedEncryptionKeysByOriginator() ([]CountByOriginator, error) {
	return countUnclaimedEncryptionKeysByOriginator(c.db)
}

func (c *conn) CountExhaustedEncryptionKeysByOriginator() ([]CountByOriginator, error) {
	return countExhaustedEncryptionKeysByOriginator(c.db)
}

func (c *conn) CountExpiredClaimedEncryptionKeysByOriginator() ([]CountByOriginator, error) {
	return countExpiredClaimedEncryptionKeysByOriginator(c.db)
}

func (c *conn) DeleteOldEncryptionKeys() (int64, error) {
	return deleteOldEncryptionKeys(c.db)
}

// ErrNoRecordWritten indicates that, though we should have been able to write
// a transaction to the DB, for some reason no record was created. This must be
// a bug with our query logic, because it should never happen.
var ErrNoRecordWritten = errors.New("we tried to write a transaction but no record was written")

var ErrKeyConsumed = errors.New("keypair has uploaded maximum number of diagnosis keys")

var ErrInvalidKeyFormat = errors.New("argument had wrong size")

var ErrDuplicateKey = errors.New("key is already registered")

var ErrInvalidOneTimeCode = errors.New("argument had wrong size")

func (c *conn) ClaimKey(oneTimeCode string, appPublicKey []byte, ctx context.Context) ([]byte, error) {
	if len(appPublicKey) != pb.KeyLength {
		return nil, ErrInvalidKeyFormat
	}
	return claimKey(c.db, oneTimeCode, appPublicKey, ctx)
}

// ErrHashIDClaimed is returned when the client tries to get a new code for a
// HashID that has already used the code
var ErrHashIDClaimed = errors.New("HashID claimed")

func (c *conn) NewKeyClaim(ctx context.Context, region, originator, hashID string) (string, error) {
	var err error

	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}

	regenerated := false

	for tries := 5; tries > 0; tries-- {

		oneTimeCode, err := generateOneTimeCode()

		if err != nil {
			return "", err
		}

		if len(hashID) == 128 {
			err = persistEncryptionKeyWithHashID(c.db, region, originator, hashID, pub, priv, oneTimeCode)
		} else {
			err = persistEncryptionKey(c.db, region, originator, pub, priv, oneTimeCode)
		}

		if err == nil {
			c.saveNewKeyClaimEvent(ctx, originator, regenerated)
			return oneTimeCode, nil
		} else if strings.Contains(err.Error(), "used hashID found") {
			return "", ErrHashIDClaimed
		} else if strings.Contains(err.Error(), "regenerate OTC for hashID") {
			regenerated = true
			log(nil, err).Warn("regenerating OTC for hashID")
			continue
		} else if strings.Contains(err.Error(), "Duplicate entry") {
			log(nil, err).Warn("duplicate one_time_code")
			continue
		} else {
			return "", err
		}
	}
	return "", err
}

func (c *conn) saveNewKeyClaimEvent(ctx context.Context, originator string, regenerated bool){

	var identifier EventType
	if regenerated {
		identifier = OTKRegenerated
	} else {
		identifier = OTKGenerated
	}

	event := Event{
		Originator: originator,
		DeviceType: Server,
		Identifier: identifier,
		Date:       time.Now(),
		Count:      1,
	}
	if err := saveEvent(c.db, event); err != nil {
		LogEvent(ctx, err, event)
	}
}

// Generate a random one time code in the format AAABBBCCCC where
// each group is made up of a character set. For each group it first
// randomizes which charater set to use. Then passes that character
// set and the desired length in another function to generate the
// string for that group.
func generateOneTimeCode() (string, error) {
	characterSets := [2][]rune{
		[]rune("AEFHJKLQRSUWXYZ"),
		[]rune("2456789"),
	}

	characterSetLength := int64(len(characterSets))

	seg1, err := rand.Int(rand.Reader, big.NewInt(characterSetLength))
	seg2, err := rand.Int(rand.Reader, big.NewInt(characterSetLength))
	seg3, err := rand.Int(rand.Reader, big.NewInt(characterSetLength))

	oneTimeCode := genRandom(characterSets[seg1.Int64()], 3) +
		genRandom(characterSets[seg2.Int64()], 3) +
		genRandom(characterSets[seg3.Int64()], 4)

	return oneTimeCode, err
}

// Generates a string of random characters based on a
// passed list of characters and a desired length. For each
// position in the desired length, generates a random number
// between 0 and the length of the character set.
func genRandom(chars []rune, length int64) string {
	var b strings.Builder
	for i := int64(0); i < length; i++ {
		nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b.WriteRune(chars[nBig.Int64()])
	}
	return b.String()
}

func (c *conn) PrivForPub(pub []byte) ([]byte, error) {
	if len(pub) != pb.KeyLength {
		return nil, ErrInvalidKeyFormat
	}
	row := privForPub(c.db, pub)
	var priv []byte
	switch err := row.Scan(&priv); err {
	case sql.ErrNoRows:
		return nil, errors.New("no record")
	case nil:
		return priv, nil
	default:
		return nil, errors.New("no record")
	}
}

func (c *conn) StoreKeys(appPubKey *[32]byte, keys []*pb.TemporaryExposureKey, ctx context.Context) error {
	return registerDiagnosisKeys(c.db, appPubKey, keys, ctx)
}

func (c *conn) FetchKeysForHours(region string, startHour uint32, endHour uint32, currentRSIN int32) ([]*pb.TemporaryExposureKey, error) {
	rows, err := diagnosisKeysForHours(c.db, region, startHour, endHour, currentRSIN)
	if err != nil {
		return nil, err
	}
	return handleKeysRows(rows)
}

func handleKeysRows(rows *sql.Rows) ([]*pb.TemporaryExposureKey, error) {
	var keys []*pb.TemporaryExposureKey
	for rows.Next() {
		var key []byte
		var rollingStartIntervalNumber int32
		var rollingPeriod int32
		var transmissionRiskLevel int32
		var region string
		err := rows.Scan(&region, &key, &rollingStartIntervalNumber, &rollingPeriod, &transmissionRiskLevel)
		if err != nil {
			return nil, err
		}
		keys = append(keys, &pb.TemporaryExposureKey{
			KeyData:                    key,
			RollingStartIntervalNumber: &rollingStartIntervalNumber,
			RollingPeriod:              &rollingPeriod,
			TransmissionRiskLevel:      &transmissionRiskLevel,
		})
	}
	return keys, nil
}

func (c *conn) CheckClaimKeyBan(identifier string) (triesRemaining int, banDuration time.Duration, err error) {
	return checkClaimKeyBan(c.db, identifier)
}

func (c *conn) ClaimKeySuccess(identifier string) error {
	return registerClaimKeySuccess(c.db, identifier)
}

func (c *conn) ClaimKeyFailure(identifier string) (int, time.Duration, error) {
	return registerClaimKeyFailure(c.db, identifier)
}

func (c *conn) DeleteOldFailedClaimKeyAttempts() (int64, error) {
	return deleteOldFailedClaimKeyAttempts(c.db)
}

func (c *conn) CountClaimedOneTimeCodes() (int64, error) {
	return countClaimedOneTimeCodes(c.db)
}

func (c *conn) CountDiagnosisKeys() (int64, error) {
	return countDiagnosisKeys(c.db)
}

func (c *conn) CountUnclaimedOneTimeCodes() (int64, error) {
	return countUnclaimedOneTimeCodes(c.db)
}

func (c *conn) Close() error {
	return c.db.Close()
}
