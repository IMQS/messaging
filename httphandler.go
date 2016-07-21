package messaging

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type sendSMSResponse struct {
	Reference         string `json:"refNumber"`
	ValidCount        int    `json:"validNumbers"`
	InvalidCount      int    `json:"invalidNumbers"`
	SendSuccess       bool   `json:"sendSuccess"`
	StatusDescription string `json:"statusDescription"`
	MessageSentCount  int    `json:"messagesSent"`
}

// StartServer is called to read the config file and initiate
// the HTTP server.
func StartServer() error {
	address := fmt.Sprintf("%v:%v", "localhost", Config.HTTPPort)
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

// HandleSendSMS should called with form-data specifying a message, and a comma-separated list of msisdns
func handleSendSMS(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userAuth, identity := userHasPermission(r)
	if userAuth != true {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}

	msg := r.FormValue("message")
	ns := r.FormValue("msisdns")

	reader := csv.NewReader(strings.NewReader(ns))
	response, err := reader.ReadAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	log.Printf("Request received from %v: send '%v' to %v recipients.", identity, msg, len(response[0]))

	cns := NormalizeMSISDNs(response[0])
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

	cleanMSISDNs := NormalizeMSISDNs(records[0])
	js, err := json.Marshal(cleanMSISDNs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
