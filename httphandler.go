package messaging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

type successJson struct {
	key string `json:"key"`
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
	router.POST("/logissue", s.handleLogIssue)

	s.Log.Infof("Messaging is listening on %v", address)
	err := http.ListenAndServe(address, router)
	if err != nil {
		s.Log.Errorf("ListenAndServe:%v\n", err)
		return err
	}
	return nil
}

func (s *MessagingServer) handleLogIssue(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	description := strings.TrimSpace(r.FormValue("description"))
	clientName := strings.TrimSpace(r.FormValue("name"))
	clientEmail := strings.TrimSpace(r.FormValue("email"))
	clientPhoneNumber := strings.TrimSpace(r.FormValue("phone"))
	callback := strings.TrimSpace(r.FormValue("callback"))
	browserName := strings.TrimSpace(r.FormValue("browser"))
	browserVersion := strings.TrimSpace(r.FormValue("browserVersion"))
	serverAddress := strings.TrimSpace(r.FormValue("serverAddress"))

	summary := fmt.Sprintf("Issue detected at %v", serverAddress)

	messageTemplate := `
	Name: %v
	Email: %v
	Phone number:%v
	Call me back: %v
	Browser: %v
	Browser version: %v
	Server: %v
	
	Message:					
	%v
	`
	message := fmt.Sprintf(messageTemplate, clientName, clientEmail, clientPhoneNumber, callback, browserName, browserVersion, serverAddress, description)

	issueKey, err := s.JiraApi.CreateIssue(summary, message)
	if err != nil {
		s.Log.Warnf("Unable to log issue with the Jira Api: %v", err.Error())
		http.Error(w, fmt.Sprintf("Unable to log issue with the Jira Api: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	ct := r.Header.Get("Content-Type")
	contentTypeList := strings.Split(ct, ";")
	contentType := contentTypeList[0]

	if contentType == "multipart/form-data" && issueKey != "" {
		statusCode, err := s.JiraApi.AddAttachments(issueKey, r)
		if err != nil {
			s.Log.Warnf("Unable to add attachments to Jira ticket with id: %v, error: %v", issueKey, err.Error())
			http.Error(w, fmt.Sprintf("Unable to add attachments to Jira ticket with id: %v, error: %v", issueKey, err.Error()), statusCode)
			return
		}

		if statusCode != http.StatusOK {
			s.Log.Warnf("Error adding attachments to Jira ticket with id: %v, status code: %v", issueKey, statusCode)
		}
	}

	result := successJson{key: issueKey}
	resultJson, err := json.Marshal(&result)
	if err != nil {
		s.Log.Warnf("Unable to marshal results into json: %v", err.Error())
		http.Error(w, fmt.Sprintf("Unable to marshal results into json: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resultJson)
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
