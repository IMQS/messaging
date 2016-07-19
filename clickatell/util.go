package clickatell

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
)

func GetContent(r *http.Response) string {
	defer r.Body.Close()
	s, _ := ioutil.ReadAll(r.Body)
	return strings.Trim(string(s), "\n")
}

func Concat(args ...string) string {
	var buffer bytes.Buffer
	for _, arg := range args {
		buffer.WriteString(arg)
	}
	return buffer.String()
}

func CreateRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		req.Header.Add("User-Agent", userAgent)
	}
	return req, err
}
