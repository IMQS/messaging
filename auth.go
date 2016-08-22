package messaging

import (
	"net/http"

	"github.com/IMQS/serviceauth"
)

// Check in the cookie whether the user that has requested the action
// has permission to do so, by calling the auth service as configured.
func userHasPermission(s *MessagingServer, r *http.Request) (bool, string) {
	if !s.Config.Authentication.Enabled {
		return true, ""
	}

	httpCode, _, authResponse := serviceauth.VerifyUserHasPermission(r, "bulksms")
	if httpCode == http.StatusOK {
		return true, authResponse.Identity
	}

	s.Log.Infof("%v: User unauthorized", httpCode)
	return false, ""
}
