package messaging

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ttacon/libphonenumber"
)

var isNumbersOnly = regexp.MustCompile(`^[0-9]*$`)

// cleanMSISDNs receives a list of MSISDNs and runs a series
// of checks to ensure that they are valid mobile numbers for the countries
// provided. Invalid and duplicate numbers are ignored and removed from the reply.
func cleanMSISDNs(ns, cs []string) []string {
	pNs := []string{}

	for _, n := range ns {
		// only allow numbers
		n = numbersOnly(n)

		// remove all spaces
		n = strings.Replace(n, " ", "", -1)

		// add the country code and verify if number is valid
		n = addCountryCode(n, cs)
		if n != "" {
			pNs = append(pNs, n)
		}
	}

	pNs = removeDuplicates(pNs)
	return pNs
}

func numbersOnly(t string) string {
	// Fast check to avoid creating a new string
	if isNumbersOnly.MatchString(t) {
		return t
	}
	// Reserve some static space for common string lengths
	cleanedSpace := [40]byte{}
	cleaned := cleanedSpace[:0]
	for _, c := range t {
		if c >= '0' && c <= '9' {
			// We know that 'c' is a single-byte character, so this type cast is safe
			cleaned = append(cleaned, byte(c))
		}
	}
	return string(cleaned)
}

// addCountryCode will find the first possible valid number for the
// given set of country codes.  This function may be a bottleneck
// for large amounts of mobile numbers. More prevalent country codes
// must be placed at the top of the country code slice to improve
// performance.
func addCountryCode(t string, cs []string) string {
	mn, _ := libphonenumber.Parse(t, cs[0])
	if libphonenumber.IsValidNumberForRegion(mn, cs[0]) {
		t = fmt.Sprintf("%v%v", *mn.CountryCode, *mn.NationalNumber)
		return t
	}
	if len(cs) > 1 {
		addCountryCode(t, cs[1:])
	}
	return ""
}

func removeDuplicates(ns []string) []string {
	fnd := map[string]bool{}
	for v := range ns {
		fnd[ns[v]] = true
	}
	res := []string{}
	for key := range fnd {
		res = append(res, key)
	}
	return res
}
