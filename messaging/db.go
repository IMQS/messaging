package messaging

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/BurntSushi/migration"
	_ "github.com/lib/pq"
)

type sqlNotifyDB struct {
	db *sql.DB
}

// DBConnection struct holds the DB configuration and can be used
// throughtout the messaging package.
type DBConnection struct {
	Driver   string
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
	SSL      bool
}

// Connect to the DB as defined in the DBConnection configuration.
func (x *DBConnection) open() (*sql.DB, error) {
	return sql.Open(x.Driver, x.connectionString(true))
}

// CreateDB takes care of creating a new DB for the Notify component
// if the DB does not yet exist.
func (x *DBConnection) createDB() error {
	log.Printf("Database does not exist, creating")
	db, err := sql.Open(x.Driver, x.connectionString(false))
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("CREATE DATABASE " + x.Database)
	if err != nil {
		return err
	}
	return nil
}

// CreateSMSData handles the DB entries for batch as well as individual
// messages after sending.
func (x *sqlNotifyDB) createSMSData(msg, eml string, resp *SMSResponse) error {
	msgs := resp.Data
	var st, stDesc string
	if resp.Error != nil {
		st = "failed"
		stDesc = resp.Error.Error()
	} else {
		st = "success"
		stDesc = ""
	}
	var id int
	// Create entry in the batchlog table and retrieve the new row ID.
	err := x.db.QueryRow(`INSERT INTO sendlog 
		(sent, originator, messagetype, quantity, success, failed, message, status, description) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		time.Now(), eml, "sms", len(msgs), 0, 0, msg, st, stDesc).Scan(&id)
	if err != nil {
		return err
	}
	// Add a new entry in the sendtransaction table for each of the SMS messages.
	for i := 0; i < len(msgs); i++ {
		_, err := x.db.Exec(`INSERT INTO sms
			(msisdn, sent, quantity, sendlogid, status, message, providerid)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			msgs[i].To, time.Now(), 1, id, msgs[i].ErrorCode, msg, msgs[i].MessageID)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateSMSData updates the SMS transaction with the retrieved status and timestamp.
func (x *sqlNotifyDB) updateSMSData(mID, stC, stD, sLogID string) error {
	_, err := x.db.Exec(`UPDATE sms SET status = $1, statustimestamp = $2 WHERE providerid = $3;`,
		stC, time.Now(), mID)
	if err != nil {
		return err
	}
	if stC == "004" { // Success
		_, err := x.db.Exec(`UPDATE sendlog SET success = success + 1 WHERE id = $1`, sLogID)
		return err
	}
	_, err = x.db.Exec(`UPDATE sendlog SET failed = failed + 1 WHERE id = $1`, sLogID)
	return err

}

// GetLastSMSID finds the most recent message that was sent to a specific
// mobile number and returns the messageID.
func (x *sqlNotifyDB) getLastSMSID(m string) (string, string, string, error) {
	var mID, sendLogID, status string
	err := x.db.QueryRow(`SELECT providerid, sendlogid, status FROM sms WHERE msisdn = $1 ORDER BY sent DESC LIMIT 1`, m).Scan(&mID, &sendLogID, &status)
	if err != nil {
		return "", "", "", errors.New("GetLastSMSID: Could not find messageID")
	}
	return mID, sendLogID, status, nil
}

// GetUnresolvedIDs finds the vendorIDs for all of the sms messages that does
// not have a valid status and that have been sent within the last period
// as specified in the i variable (in minutes).
func (x *sqlNotifyDB) getUnresolvedIDs(i int) ([][]string, error) {
	var aIDs [][]string
	rows, err := x.db.Query(`SELECT providerid, sendlogid FROM sms WHERE statustimestamp IS NULL AND sent >= $1`,
		time.Now().Add(-time.Duration(i)*time.Minute))
	if err != nil {
		return aIDs, errors.New("GetUnresolvedIDs: Could not retrieve messages")
	}
	defer rows.Close()

	for rows.Next() {
		var aID, sLogID string
		var comb []string
		if err := rows.Scan(&aID, &sLogID); err != nil {
			return aIDs, errors.New("GetUnresolvedIDs: Could not retrieve messages")
		}
		aIDs = append(aIDs, append(comb, aID, sLogID))
	}

	if err := rows.Err(); err != nil {
		return aIDs, errors.New("GetUnresolvedIDs: Could not retrieve messages")
	}
	return aIDs, nil
}

func (x *sqlNotifyDB) close() {
	if x.db != nil {
		x.db.Close()
		x.db = nil
	}
}

// RunMigrations executes the migration process.
func runMigrations(x *DBConnection) error {
	db, err := migration.Open(x.Driver, x.connectionString(true), createMigrations())
	if err == nil {
		db.Close()
	}
	return err
}

func createMigrations() []migration.Migrator {
	var migrations []migration.Migrator

	text := []string{
		`CREATE TABLE sendlog (
			id SERIAL PRIMARY KEY,
			sent TIMESTAMP,
			originator VARCHAR,
			messagetype VARCHAR,
			quantity INTEGER,
			success INTEGER,
			failed INTEGER,
			message VARCHAR,
			status VARCHAR,
			description VARCHAR
		)`,

		`CREATE TABLE sms (
			id SERIAL PRIMARY KEY, 
			msisdn VARCHAR, 
			sent TIMESTAMP, 
			quantity SMALLINT, 
			sendlogid INTEGER REFERENCES sendlog (id),
			status VARCHAR,
			statustimestamp TIMESTAMP,
			message VARCHAR,
			providerid VARCHAR
			)`,
	}

	for _, src := range text {
		srcCapture := src
		migrations = append(migrations, func(tx migration.LimitedTx) error {
			_, err := tx.Exec(srcCapture)
			return err
		})
	}
	return migrations
}

func (x *DBConnection) connectionString(addDB bool) string {
	sslmode := "disable"
	if x.SSL {
		sslmode = "require"
	}
	conStr := fmt.Sprintf("host=%v user=%v password=%v sslmode=%v", x.Host, x.User, x.Password, sslmode)
	if addDB {
		conStr += fmt.Sprintf(" dbname=%v", x.Database)
	}
	if x.Port != 0 {
		conStr += fmt.Sprintf(" port=%v", x.Port)
	}
	return conStr
}
