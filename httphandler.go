package messaging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type sendSMSResponse struct {
	RefNumber         string `json:"refNumber"`
	ValidNumbers      int    `json:"validNumbers"`
	InvalidNumbers    int    `json:"invalidNumbers"`
	SendSuccess       bool   `json:"sendSuccess"`
	StatusDescription string `json:"statusDescription"`
	MessagesSent      int    `json:"messagesSent"`
}

type SMSRequest struct {
	MSISDNS []string `json:"msisdns"`
	Message string   `json:"message"`
}

const smsCharLength = 160

// StartServer is called to read the config file and initiate
// the HTTP server.
func (s *MessagingServer) StartServer() error {
	address := fmt.Sprintf(":%v", s.Config.HTTPPort)
	router := httprouter.New()
	router.GET("/messagestatus/:msisdn", s.handleMessageStatus)
	router.GET("/ping", s.handlePing)
	router.POST("/sendsms", s.handleSendSMS)
	router.POST("/normalize", s.handleNormalize)

	s.Log.Infof("Messaging is listening on %v", address)
	err := http.ListenAndServe(address, router)
	if err != nil {
		s.Log.Errorf("ListenAndServe:%v\n", err)
		return err
	}
	return nil
}

// HandleSendSMS should called with form-data specifying a message, and a comma-separated list of msisdns.
// It can be expanded to accept a JSON object containing fields such as name, surname, age, etc.  These
// can then be replaced in the message before sending to allow for personalized messages.
func (s *MessagingServer) handleSendSMS(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userAuth, identity := userHasPermission(s, r)
	if userAuth != true {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}

	var postData SMSRequest
	err := json.NewDecoder(r.Body).Decode(&postData)

	if err != nil {
		http.Error(w, "Invalid message or msisdn json data", http.StatusNotAcceptable)
		return
	}

	if len(postData.Message) == 0 || len(postData.MSISDNS) == 0 {
		http.Error(w, "Invalid message or msisdn data", http.StatusNotAcceptable)
		return
	}

	// Strip out any non-ascii characters from the message that could result in
	// multiple messages being sent.
	cleanMsg := allowOnlyASCII(postData.Message)

	// SMS with 7 bit character encoding messages are limited to a lenght of 160 characters.
	// Check if message fits into one message (segements * sms length)
	if len(cleanMsg) > s.Config.SMSProvider.MaxMessageSegments*smsCharLength {
		lErr := fmt.Sprintf("Message exceeds max allowed length (%v characters)", s.Config.SMSProvider.MaxMessageSegments*smsCharLength)
		http.Error(w, lErr, http.StatusNotAcceptable)
		return
	}

	s.Log.Debugf("Request received from %v: send '%v' to %v recipients.", identity, cleanMsg, len(postData.MSISDNS))

	cns := cleanMSISDNs(postData.MSISDNS, s.Config.SMSProvider.Countries)
	sendID, err := s.SendSMSMessages(cleanMsg, identity, cns)

	sendR := sendSMSResponse{
		RefNumber:         sendID,
		ValidNumbers:      len(cns),
		InvalidNumbers:    len(postData.MSISDNS) - len(cns),
		SendSuccess:       false,
		StatusDescription: "",
		MessagesSent:      0,
	}
	if err == nil {
		sendR.SendSuccess = true
		sendR.MessagesSent = len(cns) // Assuming sending only one message per MSISDN
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
func (s *MessagingServer) handleMessageStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userAuth, _ := userHasPermission(s, r)
	if userAuth != true {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}

	n := ps.ByName("msisdn")
	st, err := s.GetNumberStatus(n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%v", st)
}

// HandleNormalize expects a comma separated list of mobile numbers which it would
// then run through a series of operations to validate, clean up and remove
// duplicates.  It returns a JSON list of valid numbers.
func (s *MessagingServer) handleNormalize(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userAuth, _ := userHasPermission(s, r)
	if userAuth != true {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}

	var postData SMSRequest
	err := json.NewDecoder(r.Body).Decode(&postData)

	if err != nil {
		http.Error(w, "Invalid message or msisdn json data", http.StatusNotAcceptable)
		return
	}

	cleanMSISDNs := cleanMSISDNs(postData.MSISDNS, s.Config.SMSProvider.Countries)
	js, err := json.Marshal(cleanMSISDNs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (s *MessagingServer) handlePing(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "{\"Timestamp\": %v}", time.Now().Unix())
}
