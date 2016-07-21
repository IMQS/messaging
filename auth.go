package messaging

import (
	"log"
	"net/http"

	"github.com/IMQS/serviceauth"
)

// Check in the cookie whether the user that has requested the action
// has permission to do so, by calling the auth service as configured.
func userHasPermission(r *http.Request) (bool, string) {
	if !Config.Authentication.Enabled {
		return true, ""
	}

	switch Config.Authentication.Service {
	case "serviceauth":
		return serviceAuthPermission(r)
	}
	return false, ""
}

func serviceAuthPermission(r *http.Request) (bool, string) {
	httpCode, _, authResponse := serviceauth.VerifyUserHasPermission(r, "bulksms")
	if httpCode == http.StatusOK {
		identity := authResponse.Identity
		return true, identity
	}

	log.Printf("%v: User unauthorized", httpCode)
	return false, ""
}
