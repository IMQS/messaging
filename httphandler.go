package messaging

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/julienschmidt/httprouter"
)

// CR: Why make the variable names inside the Go code different to the JSON variable names?
// I understand if you want to keep the JSON values camelCase, but at least make the
// variable names identical, aside from case.
type sendSMSResponse struct {
	Reference         string `json:"refNumber"`
	ValidCount        int    `json:"validNumbers"`
	InvalidCount      int    `json:"invalidNumbers"`
	SendSuccess       bool   `json:"sendSuccess"`
	StatusDescription string `json:"statusDescription"`
	MessageSentCount  int    `json:"messagesSent"`
}

const smsCharLength = 160

// StartServer is called to read the config file and initiate
// the HTTP server.
func StartServer() error {
	address := fmt.Sprintf(":%v", Config.HTTPPort)
	router := httprouter.New()
	router.GET("/messageStatus/:msisdn", handleMessageStatus)
	router.POST("/sendSMS", handleSendSMS)
	router.POST("/normalize", handleNormalize)

	log.Printf("Messaging is listening on %v", address)
	err := http.ListenAndServe(address, router)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
		return err
	}
	return nil
}

// HandleSendSMS should called with form-data specifying a message, and a comma-separated list of msisdns.
// It can be expanded to accept a JSON object containing fields such as name, surname, age, etc.  These
// can then be replaced in the message before sending to allow for personalized messages.
func handleSendSMS(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userAuth, identity := userHasPermission(r)
	if userAuth != true {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}

	msg := r.FormValue("message")
	// CR: I think 'numbers' is probably a better variable name, and likewise 'cleanedNumbers'
	ns := r.FormValue("msisdns")

	if len(msg) == 0 || len(ns) == 0 {
		http.Error(w, "Invalid message or msisdn data", http.StatusNotAcceptable)
		return
	}

	// CR: Probably simpler to just use strings.Split(ns, ",") here
	reader := csv.NewReader(strings.NewReader(ns))
	response, err := reader.ReadAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	// Check if message fits into one message (segements * sms length)
	// CR: I'm pretty sure SMS limits are imposed on encoded length, not number of runes. Not sure
	// about exact encodings etc... thought it was some variant of UTF16 when necessary, but I don't know for sure.
	// But pretty sure counting runes is wrong.
	// Although.. we are imposing this limit here. It guess it's fine whatever it is, but we should at least
	// make it clear that this limit is arbitrary, and imposed by us.
	if utf8.RuneCountInString(msg) > Config.SMSProvider.MaxMessageSegments*smsCharLength {
		lErr := fmt.Sprintf("Message exceeds max allowed length (%v characters)", Config.SMSProvider.MaxMessageSegments*smsCharLength)
		http.Error(w, lErr, http.StatusNotAcceptable)
		return
	}

	log.Printf("Request received from %v: send '%v' to %v recipients.", identity, msg, len(response[0]))

	cns := cleanMSISDNs(response[0], Config.SMSProvider.Countries)
	sendID, err := SendSMS(msg, identity, cns)

	sendR := sendSMSResponse{
		Reference:         sendID,
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
func handleMessageStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userAuth, _ := userHasPermission(r)
	if userAuth != true {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}

	n := ps.ByName("msisdn")
	st, err := GetNumberStatus(n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// CR: Should probably set Content-Type: text/plain
	fmt.Fprintf(w, "%v", st)
}

// HandleNormalize expects a comma separated list of mobile numbers which it would
// then run through a series of operations to validate, clean up and remove
// duplicates.  It returns a JSON list of valid numbers.
func handleNormalize(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userAuth, _ := userHasPermission(r)
	if userAuth != true {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}
	msisdns := r.FormValue("msisdns")
	reader := csv.NewReader(strings.NewReader(msisdns))
	records, _ := reader.ReadAll()

	cleanMSISDNs := cleanMSISDNs(records[0], Config.SMSProvider.Countries)
	js, err := json.Marshal(cleanMSISDNs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
