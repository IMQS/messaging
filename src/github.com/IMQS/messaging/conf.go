package messaging

import (
	"encoding/json"
	"log"
	"os"
)

/*

Sample config:

{
	"HTTPPort": 2012,
	"smsProvider": {
		"name": "Clickatell",
		"enabled": true,
		"token": "123abc",
		"maxMessageSegments": 1,
		"maxBatchSize": 600,
		"countries": ["ZA", "BW", "US"]
	},
	"authentication": {
		"service": "serviceauth",
		"enabled": true
	},
	"deliveryStatus": {
		"enabled": true,
		"updateInverval": 15
	},
	"dbConnection": {
		"Driver": "postgres",
		"Host": "localhost",
		"Port": 5432,
		"Database": "messaging",
		"User": "jim",
		"Password": "123",
		"SSL": false
	}
}

*/

// Variables used throughout the messaging package
var (
	Config = configuration{}
	DB     sqlNotifyDB
	DBCon  dbConnection
)

type configuration struct {
	HTTPPort       int
	SMSProvider    smsProvider
	Authentication authConfig
	DeliveryStatus deliveryInterval
	DBConnection   dbConnection
}

type smsProvider struct {
	Name               string
	Enabled            bool
	Token              string
	MaxMessageSegments int
	MaxBatchSize       int
	Countries          []string
	Custom1            string
	Custom2            string
	Custom3            string
}

type authConfig struct {
	Service string
	Enabled bool
}

type dbConnection struct {
	Driver   string
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
	SSL      bool
}

type deliveryInterval struct {
	Enabled        bool
	UpdateInverval int
}

// NewConfig reads the config file, opens the DB
// and starts the interval ticker.
func NewConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)

	if err = decoder.Decode(&Config); err != nil {
		log.Println("Error parsing config file:", err)
		return err
	}
	DBCon := Config.DBConnection

	if DB.db, err = DBCon.open(); err != nil {
		log.Printf("Error connecting to Messaging DB: %v", err)
		return err
	}

	if err := DB.db.Ping(); err != nil {
		if err = DBCon.createDB(); err != nil {
			log.Printf("DB Create: %v", err)
			return err
		}
	}

	if err := runMigrations(&DBCon); err != nil {
		return err
	}
	startInterval()
	return nil
}
