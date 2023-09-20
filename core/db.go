package messaging

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/BurntSushi/migration"
	_ "github.com/lib/pq"
)

type sqlNotifyDB struct {
	db *sql.DB
}

// CreateSMSData handles the DB entries for batch as well as individual
// messages after sending.
func (x *sqlNotifyDB) createSMSData(messageText, email string, messages []SendSMSResponseMessage, err error) (string, error) {
	var st, stDesc string
	if err != nil {
		st = "failed"
		stDesc = err.Error()
	} else {
		st = "success"
		stDesc = ""
	}
	var id int
	// Create entry in the batchlog table and retrieve the new row ID.
	err = x.db.QueryRow(`INSERT INTO sendlog 
		(senttime, originator, type, quantity, delivered, failed, sent, message, status, description) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
		time.Now().UTC(), email, "sms", len(messages), 0, 0, len(messages), messageText, st, stDesc).Scan(&id)
	if err != nil {
		return "", err
	}
	// Add a new entry in the sendtransaction table for each of the SMS messages.
	for i := 0; i < len(messages); i++ {
		msgStatus := Sent
		if messages[i].ErrorCode != "0" {
			msgStatus = messages[i].ErrorCode
		}
		_, err := x.db.Exec(`INSERT INTO sms
			(msisdn, senttime, segments, sendlogid, status, message, providerid)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			messages[i].To, time.Now().UTC(), messages[i].Segments, id, msgStatus, messageText, messages[i].MessageID)
		if err != nil {
			return "", err
		}
	}
	return strconv.Itoa(id), nil
}

// UpdateSMSData updates the SMS transaction with the retrieved status and timestamp.
func (x *sqlNotifyDB) updateSMSData(messageID, statusCode, statusDescription, sendLogID string, segments int) (err error) {
	if statusCode != Sent {
		_, err := x.db.Exec(`UPDATE sms SET status = $1, statustimestamp = $2, segments = $3 WHERE providerid = $4;`,
			statusCode, time.Now().UTC(), segments, messageID)
		if err != nil {
			return err
		}
	}

	if statusCode == Delivered { // Success
		_, err := x.db.Exec(`UPDATE sendlog SET delivered = delivered + 1, sent = sent - 1 WHERE id = $1`, sendLogID)
		return err
	} else if statusCode == Failed {
		_, err := x.db.Exec(`UPDATE sendlog SET failed = failed + 1, sent = sent - 1 WHERE id = $1`, sendLogID)
		return err
	}
	return err

}

// GetLastSMSID finds the most recent message that was sent to a specific
// mobile number and returns the messageID.
func (x *sqlNotifyDB) getLastSMSID(m string) (messageID, sendLogID, status string, err error) {
	err = x.db.QueryRow(`SELECT providerid, sendlogid, status FROM sms WHERE msisdn = $1 ORDER BY senttime DESC LIMIT 1`, m).Scan(&messageID, &sendLogID, &status)
	if err != nil {
		return "", "", "", errors.New("GetLastSMSID: Could not find messageID")
	}
	return messageID, sendLogID, status, nil
}

// GetUnresolvedIDs finds the vendorIDs for all of the sms messages that does
// not have a valid status and that have been sent within the last period
// as specified in the i variable (in minutes).
func (x *sqlNotifyDB) getUnresolvedIDs(s string) (vendorIDs [][]string, err error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return vendorIDs, err
	}
	rows, err := x.db.Query(`SELECT providerid, sendlogid FROM sms WHERE statustimestamp IS NULL AND senttime >= $1`,
		time.Now().UTC().Add(-d))
	if err != nil {
		return vendorIDs, errors.New("GetUnresolvedIDs: Could not retrieve messages")
	}
	defer rows.Close()

	for rows.Next() {
		var aID, sLogID string
		var comb []string
		if err := rows.Scan(&aID, &sLogID); err != nil {
			return vendorIDs, errors.New("GetUnresolvedIDs: Could not retrieve messages")
		}
		vendorIDs = append(vendorIDs, append(comb, aID, sLogID))
	}

	if err := rows.Err(); err != nil {
		return vendorIDs, errors.New("GetUnresolvedIDs: Could not retrieve messages")
	}
	return vendorIDs, nil
}

func (x *sqlNotifyDB) close() {
	if x.db != nil {
		x.db.Close()
		x.db = nil
	}
}

///////////////////////////////////////////////////////////////////////////////

// Connect to the DB as defined in the dbConnection configuration.
func (x *ConfigDBConnection) open() (*sql.DB, error) {
	return sql.Open(x.Driver, x.connectionString(true))
}

// CreateDB takes care of creating a new DB for the Notify component
// if the DB does not yet exist.
func (x *ConfigDBConnection) createDB() error {
	messagingDB := x.Database
	x.Database = "postgres" // Connect to the postgres DB when creating a new database
	db, err := sql.Open(x.Driver, x.connectionString(true))
	if err != nil {
		return err
	}
	x.Database = messagingDB
	defer db.Close()
	_, err = db.Exec("CREATE DATABASE " + x.Database)
	if err != nil {
		return err
	}
	return nil
}

// RunMigrations executes the migration process.
func (s *MessagingServer) runMigrations() error {
	db, err := migration.Open(s.Config.DBConnection.Driver, s.Config.DBConnection.connectionString(true), createMigrations())
	if err == nil {
		db.Close()
	}
	return err
}

// A new 'sendlog' entry is created in the table for each batch of messages that are submitted,
// where the message is the same for all recipients. A new entry is created in the 'sms' table for
// each message that is sent to a unique msisdn, and is linked to the 'sendlog' table by the
// 'sendlogid' field.
func createMigrations() []migration.Migrator {
	var migrations []migration.Migrator

	text := []string{
		`CREATE TABLE sendlog (
			id BIGSERIAL PRIMARY KEY,
			senttime TIMESTAMP,
			originator VARCHAR,
			type VARCHAR,
			quantity INTEGER,
			delivered INTEGER,
			failed INTEGER,
			sent INTEGER,
			message VARCHAR,
			status VARCHAR,
			description VARCHAR
		)`,

		`CREATE TABLE sms (
			id BIGSERIAL PRIMARY KEY, 
			msisdn VARCHAR, 
			senttime TIMESTAMP, 
			segments SMALLINT, 
			sendlogid INTEGER,
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

func (x *ConfigDBConnection) connectionString(addDB bool) string {
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
