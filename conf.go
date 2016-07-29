package messaging

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/IMQS/log"
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

type MessagingServer struct {
	Config   Configuration
	Log      *log.Logger
	DB       sqlNotifyDB
	Interval IntervalService
}

type Configuration struct {
	HTTPPort       int
	Logfile        string
	SMSProvider    ConfigSmsProvider
	Authentication ConfigAuth
	DeliveryStatus ConfigDeliveryInterval
	DBConnection   ConfigDBConnection
}

type ConfigSmsProvider struct {
	Name               string
	Enabled            bool
	Token              string
	MaxMessageSegments int
	MaxBatchSize       int
	Countries          []string
}

type ConfigAuth struct {
	Service string
	Enabled bool
}

type ConfigDBConnection struct {
	Driver   string
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
	SSL      bool
}

// ConfigDeliveryInterval controls the behaviour of the delivery status checker.
type ConfigDeliveryInterval struct {
	Enabled        bool
	UpdateInterval string
}

// Initialize opens a log file, opens the DB and starts the interval ticker.
func (s *MessagingServer) Initialize() error {
	var err error

	s.Log = log.New(s.Config.Logfile)
	s.Log.Level = 0
	s.DB.db, err = s.Config.DBConnection.open()
	if err != nil {
		s.Log.Errorf("Error connecting to Messaging DB: %v", err)
		return err
	}

	if err = s.DB.db.Ping(); err != nil {
		s.Log.Infof("Database does not exist, creating")
		if err = s.Config.DBConnection.createDB(); err != nil {
			s.Log.Errorf("DB Create: %v", err)
			return err
		}
	}

	if err = s.runMigrations(); err != nil {
		return err
	}
	s.startInterval()
	return nil
}

// NewConfig reads the config file
func (c *Configuration) NewConfig(filename string) error {

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)

	if err = decoder.Decode(&c); err != nil {
		fmt.Println("Error parsing config file:", err)
		return err
	}

	return nil
}
