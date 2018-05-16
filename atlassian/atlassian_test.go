package atlassian

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func generateSuccessResponse() *SuccessResponse {
	issueId := seededRand.Intn(9999)
	issueKey := "AB-" + strconv.Itoa(issueId)
	return &SuccessResponse{
		Id:   strconv.Itoa(issueId),
		Key:  issueKey,
		Self: "/" + issueKey,
	}
}

func generateErrorResponse() *ErrorResponse {
	messages := make([]string, 0)
	messages = append(messages, "Input not in correct format")
	return &ErrorResponse{
		ErrorMessages: messages,
	}
}

func TestCreateIssue(t *testing.T) {
	jiraTestRestApi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issue := &Issue{}
		err := json.NewDecoder(r.Body).Decode(&issue)
		if err != nil {
			t.Log(err)
		}

		reqBody, err := json.Marshal(issue)
		if err != nil {
			t.Log(err)
		}

		if len(reqBody) > 0 {
			respSuccess := generateSuccessResponse()
			respSuccessData, _ := json.Marshal(respSuccess)
			w.WriteHeader(http.StatusOK)
			w.Write(respSuccessData)
		} else {
			respError := generateErrorResponse()
			respErrorData, _ := json.Marshal(respError)
			w.WriteHeader(http.StatusBadRequest)
			w.Write(respErrorData)
		}
	}))

	config := &ConfigJiraProvider{
		Endpoint:   jiraTestRestApi.URL,
		Username:   "test-user",
		Password:   "test-token",
		ProjectId:  "1234",
		ProjectKey: "AB",
		Assignee:   "testassignee",
	}

	jiraTestApi := NewImqsJiraApi(config)

	testIssueSummary := "Test summary"
	testIssueDescription := "This is a test description"

	issuedTicketKey, err := jiraTestApi.CreateIssue(testIssueSummary, testIssueDescription)
	if err != nil {
		t.Fail()
		fmt.Printf("Error : %v\n", err.Error())
	}

	if len(issuedTicketKey) <= 0 {
		t.Fail()
		fmt.Printf("Invalid Jira ticket number: %v\n", err.Error())
	}

	t.Logf("Issued Jira ticket number: %v", issuedTicketKey)
	fmt.Printf("Issued Jira ticket number: %v\n", issuedTicketKey)
}
