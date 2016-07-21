package messaging

import (
	"encoding/json"
	"log"
	"os"
)

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
	Name         string
	Enabled      bool
	Token        string
	MaxBatchSize int
	Custom1      string
	Custom2      string
	Custom3      string
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
