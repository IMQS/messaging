package messaging

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/IMQS/serviceauth"
	"github.com/julienschmidt/httprouter"
)

type configuration struct {
	HTTPPort       int
	SMSEnabled     bool
	SMSProvider    smsProvider
	DeliveryStatus DeliveryInterval
	DBConnection   DBConnection
}

type sendSMSResponse struct {
	ValidCount        int    `json:"validMsisdns"`
	InvalidCount      int    `json:"invalidMsisdns"`
	SendSuccess       bool   `json:"sendSuccess"`
	StatusDescription string `json:"statusDescription"`
	MessageSentCount  int    `json:"messagesSent"`
}

// Variables used throughout the messaging package
var (
	Config = configuration{}
	DB     sqlNotifyDB
	DBCon  DBConnection
)

// HandleSendSMS should called with form-data specifying a message, and a comma-separated list of msisdns
func HandleSendSMS(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if userHasPermission(r) != true {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}

	Bootstrap()

	msg := r.FormValue("message")
	ns := r.FormValue("msisdns")

	// GET USER FROM COOKIE?
	eml := "test@imqs.co.za"

	reader := csv.NewReader(strings.NewReader(ns))
	response, err := reader.ReadAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	log.Printf("Request received from %v: send '%v' to %v recipients.", eml, msg, len(response[0]))

	cns := NormalizeMSISDNs(response[0])
	err = SendSMS(msg, eml, cns)

	sendR := sendSMSResponse{
		ValidCount:        len(cns),
		InvalidCount:      len(response[0]) - len(cns),
		SendSuccess:       false,
		StatusDescription: "",
		MessageSentCount:  0,
	}
	if err == nil {
		sendR.SendSuccess = true
		sendR.MessageSentCount = len(cns) // Assuming sending only one message per MSISDN
	} else {
		sendR.StatusDescription = err.Error()
	}
	js, err := json.Marshal(sendR)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// HandleMessageStatus retrieves the delivery status for the last message delivered
// to a mobile number.
func HandleMessageStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if userHasPermission(r) != true {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}
	Bootstrap()

	n := ps.ByName("msisdn")
	st, err := GetNumberStatus(n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%v", st)
}

// HandleNormalize expects a comma separated list of mobile numbers which it would
// then run through a series of operations to validate, clean up and remove
// duplicates.  It returns a JSON list of valid numbers.
func HandleNormalize(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if userHasPermission(r) != true {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}
	Bootstrap()
	msisdns := r.FormValue("msisdns")
	reader := csv.NewReader(strings.NewReader(msisdns))
	records, _ := reader.ReadAll()

	cleanMSISDNs := NormalizeMSISDNs(records[0])
	js, err := json.Marshal(cleanMSISDNs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// Bootstrap reads config, setups DB and starts ticker
func Bootstrap() error {
	// Read config file if not already done
	if (configuration{}) == Config {
		file, err := os.Open("messaging/conf.json")
		if err != nil {
			log.Fatal("Open:", err)
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
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////

// Check in the cookie whether the user that has requested the action
// has permission to do so, by calling the serviceauth package.
func userHasPermission(r *http.Request) bool {
	// REMOVE THIS CODE ONCE TESTING IS COMPLETE:
	return true

	httpCode, _, d := serviceauth.VerifyUserHasPermission(r, "bulksms")
	fmt.Println("UserPermission response =", d)
	if httpCode == http.StatusOK {
		return true
	}

	log.Printf("SendSMS attempt: User unauthorized, %v", d)
	return false
}
