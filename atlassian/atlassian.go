package atlassian

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
)

type SuccessResponse struct {
	Id   string `json:"id,omitempty"`
	Key  string `json:"key,omitempty"`
	Self string `json:"self,omitempty"`
}

type ErrorResponse struct {
	ErrorMessages []string    `json:"errorMessages,omitempty"`
	Errors        interface{} `json:"errors,omitempty"`
}

type IssueRequest struct {
	Name        string
	Email       string
	Phone       string
	Description string
	Summary     string
}

type Project struct {
	Id string `json:"id,omitempty"`
}

type IssueType struct {
	Id string `json:"id,omitempty"`
}

type Priority struct {
	Id string `json:"id,omitempty"`
}

type Assignee struct {
	Name string `json:"name,omitempty"`
}

type Reporter struct {
	Name string `json:"name,omitempty"`
}

type Fields struct {
	Project          *Project           `json:"project,omitempty"`
	Summary          string             `json:"summary,omitempty"`
	IssueType        *IssueType         `json:"issuetype,omitempty"`
	Assignee         *Assignee          `json:"assignee,omitempty"`
	Reporter         *Reporter          `json:"reporter,omitempty"`
	Priority         *Priority          `json:"priority,omitempty"`
	Environment      string             `json:"environment,omitempty"`
	Description      string             `json:"description,omitempty"`
	StepsToReplicate string             `json:"customfield_11600,omitempty"`
	Environments     []*IMQSEnvironment `json:"customfield_12004,omitempty"`
	Projects         []*IMQSProject     `json:"customfield_12301,omitempty"`
	Product          *IMQSProduct       `json:"customfield_11301,omitempty"`
}
type IMQSProject struct {
	Id string `json:"id,omitempty"`
}

type IMQSProduct struct {
	Id string `json:"id,omitempty"`
}

type IMQSEnvironment struct {
	Id string `json:"id,omitempty"`
}

type Issue struct {
	Fields *Fields `json:"fields"`
}

// New issue Will have pre-selected defaults
// Service Desk personnel will have to go through all the properties and change to the right ones
// depending on the query
func NewIssue(summary, description, jiraProjectId, assignee string) *Issue {
	projects := make([]*IMQSProject, 0)
	environments := make([]*IMQSEnvironment, 0)

	// Default - "IMQS Internal tools"
	internalProject := &IMQSProject{
		Id: "13501",
	}
	projects = append(projects, internalProject)

	// Default - "Production"
	prodEnvironment := &IMQSEnvironment{
		Id: "12617",
	}
	environments = append(environments, prodEnvironment)

	fields := &Fields{
		Project: &Project{
			Id: jiraProjectId,
		},
		Summary: summary,

		// Issue type - Production bug
		IssueType: &IssueType{
			Id: "11303",
		},
		Assignee: &Assignee{
			Name: assignee,
		},
		Reporter: &Reporter{
			Name: assignee,
		},

		// Priority - Normal level
		Priority: &Priority{
			Id: "3",
		},
		Description:      description,
		StepsToReplicate: description,
		Environments:     environments,
		Projects:         projects,

		// Product default to "IMQS Tools"
		Product: &IMQSProduct{
			Id: "11406",
		},
	}
	return &Issue{
		Fields: fields,
	}
}

type ImqsJiraApi struct {
	Config *ConfigJiraProvider
}

type ConfigJiraProvider struct {
	Endpoint   string
	Username   string
	Password   string
	ProjectId  string
	Assignee   string
	ProjectKey string
}

func NewImqsJiraApi(cfg *ConfigJiraProvider) *ImqsJiraApi {
	return &ImqsJiraApi{
		Config: cfg,
	}
}

func (js *ImqsJiraApi) CreateIssue(summary, description string) (string, error) {
	issue := NewIssue(summary, description, js.Config.ProjectId, js.Config.Assignee)
	issueData, err := json.Marshal(issue)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", js.Config.Endpoint+"/issue", bytes.NewBuffer(issueData))
	if err != nil {
		return "", err
	}

	authToken := base64.StdEncoding.EncodeToString([]byte(js.Config.Username + ":" + js.Config.Password))

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+authToken)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	responseDetails, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	successResponse := SuccessResponse{}
	err = json.Unmarshal(responseDetails, &successResponse)
	if err != nil {
		return "", err
	}

	return successResponse.Key, nil
}

func (js *ImqsJiraApi) AddAttachments(issueKey string, r *http.Request) (int, error) {
	messageBuffer := new(bytes.Buffer)
	mpWriter := multipart.NewWriter(messageBuffer)

	for _, files := range r.MultipartForm.File {
		for _, file := range files {
			var infile multipart.File

			// Read file data into memory
			infile, err := file.Open()
			if err != nil {
				return http.StatusBadRequest, err
			}

			// All files must have fieldname 'file' when calling the Jira Api
			// For more details see: https://developer.atlassian.com/cloud/jira/platform/rest/#api-api-2-issue-issueIdOrKey-attachments-post
			part, err := mpWriter.CreateFormFile("file", file.Filename)
			if err != nil {
				return http.StatusBadRequest, err
			}

			// Copy file data to form
			_, err = io.Copy(part, infile)
			if err != nil {
				return http.StatusBadRequest, err
			}
		}
	}

	err := mpWriter.Close()
	if err != nil {
		return http.StatusBadRequest, err
	}

	req, err := http.NewRequest("POST", js.Config.Endpoint+"/issue/"+issueKey+"/attachments", messageBuffer)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	authToken := base64.StdEncoding.EncodeToString([]byte(js.Config.Username + ":" + js.Config.Password))

	req.Header.Add("Authorization", "Basic "+authToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", mpWriter.FormDataContentType())

	// In order to protect against XSRF attacks and because the "Add Attachment" method accepts multipart/form-data, it has XSRF protection on it.
	// It is a requirement that you must submit a header of the form "X-Atlassian-Token: no-check" with the request, otherwise it will be blocked.
	// For more details see: https://developer.atlassian.com/cloud/jira/platform/rest/#api-api-2-issue-issueIdOrKey-attachments-post
	req.Header.Set("X-Atlassian-Token", "no-check")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	defer response.Body.Close()

	return response.StatusCode, nil
}
