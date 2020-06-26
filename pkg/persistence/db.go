package persistence

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	pb "github.com/CovidShield/server/pkg/proto/covidshield"

	"github.com/Shopify/goose/logger"
	// inject mysql support for database/sql
	_ "github.com/go-sql-driver/mysql"
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
	FetchKeysForDateNumber(string, uint32, int32) ([]*pb.TemporaryExposureKey, error)
	StoreKeys(*[32]byte, []*pb.TemporaryExposureKey) error
	NewKeyClaim(string, string, string) (string, error)
	CheckHashID(string) (int64, error)
	ClaimKey(string, []byte) ([]byte, error)
	PrivForPub([]byte) ([]byte, error)

	CheckClaimKeyBan(string) (triesRemaining int, banDuration time.Duration, err error)
	ClaimKeySuccess(string) error
	ClaimKeyFailure(string) (triesRemaining int, banDuration time.Duration, err error)

	DeleteOldDiagnosisKeys() (int64, error)
	DeleteOldEncryptionKeys() (int64, error)
	DeleteOldFailedClaimKeyAttempts() (int64, error)

	CountClaimedKeys() (int64, error)
	CountDiagnosisKeys() (int64, error)
	CountUnclaimedKeys() (int64, error)

	Close() error
}

type conn struct {
	db *sql.DB
}

var log = logger.New("db")

const (
	// TODO: adjust these to deployment and source them from env
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
	db, err := sql.Open("mysql", url)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(maxConnLifetime)
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	return &conn{db: db}, nil
}

func (c *conn) DeleteOldDiagnosisKeys() (int64, error) {
	return deleteOldDiagnosisKeys(c.db)
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

func (c *conn) ClaimKey(oneTimeCode string, appPublicKey []byte) ([]byte, error) {
	if len(appPublicKey) != pb.KeyLength {
		return nil, ErrInvalidKeyFormat
	}
	return claimKey(c.db, oneTimeCode, appPublicKey)
}

const maxOneTimeCode = 1e8

func (c *conn) NewKeyClaim(region, originator, hashID string) (string, error) {
	var err error
	var n *big.Int

	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}

	for tries := 5; tries > 0; tries-- {
		n, err = rand.Int(rand.Reader, big.NewInt(maxOneTimeCode)) // [0,max)
		if err != nil {
			return "", err
		}

		oneTimeCode := fmt.Sprintf("%08d", n)

		err = persistEncryptionKey(c.db, region, originator, hashID, pub, priv, oneTimeCode)
		if err == nil {
			return oneTimeCode, nil
		} else if strings.Contains(err.Error(), "Duplicate entry") {
			log(nil, err).Warn("duplicate one_time_code")
		} else {
			return "", err
		}
	}
	return "", err
}

func (c *conn) CheckHashID(hashID string) (int64, error) {
	return checkHashID(c.db, hashID)
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

func (c *conn) StoreKeys(appPubKey *[32]byte, keys []*pb.TemporaryExposureKey) error {
	return registerDiagnosisKeys(c.db, appPubKey, keys)
}

func (c *conn) FetchKeysForDateNumber(region string, dateNumber uint32, currentRSIN int32) ([]*pb.TemporaryExposureKey, error) {
	rows, err := diagnosisKeysForDateNumber(c.db, region, dateNumber, currentRSIN)
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

func (c *conn) CountClaimedKeys() (int64, error ) {
	return countClaimedKeys(c.db)
}

func (c *conn) CountDiagnosisKeys() (int64, error ) {
	return countDiagnosisKeys(c.db)
}

func (c *conn) CountUnclaimedKeys() (int64, error ) {
	return countClaimedKeys(c.db)
}

func (c *conn) Close() error {
	return c.db.Close()
}
