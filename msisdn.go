package messaging

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/ttacon/libphonenumber"
)

// CR:
var isNumbersOnly = regexp.MustCompile(`^[0-9]+$`)

// cleanMSISDNs receives a list of MSISDNs and runs a series
// of checks to ensure that they are valid mobile numbers for the countries
// provided. Invalid and duplicate numbers are ignored and removed from the reply.
func cleanMSISDNs(ns, cs []string) []string {
	pNs := []string{}
	// CR: I'd say that this is crazy overkill. I can't imagine that phone number
	// cleaning is so slow as to warrant the extra complexity of launching
	// multiple threads.
	workers := make(chan string, len(ns)/8)
	var wg sync.WaitGroup

	// Start 8 worker go routines for concurrency
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			for gn := range workers {
				// only allow numbers
				numbersOnly(&gn)

				// remove all spaces
				gn = strings.Replace(gn, " ", "", -1)

				// add the country code and verify if number is valid
				addCountryCode(&gn, cs)
				if gn != "" {
					pNs = append(pNs, gn)
				}
			}
			wg.Done()
		}()
	}

	for _, n := range ns {
		workers <- n
	}
	close(workers)
	wg.Wait()
	pNs = removeDuplicates(pNs)
	return pNs
}

// CR: Because strings are immutable in Go, there is no performance benefit to sending in *string.
// It's just as fast to make this function's definition
//   func numbersOnly(t string) string
// and such a design is generally more readable.
// Another thing -- this seems very dubious -- ie iterating over a string,
// while mutating it's contents. I definitely don't think this is healthy.
// What I would do instead is this:
/*
func numbersOnly2(t string) string {
	// Fast check to avoid creating a new string
	if isNumbersOnly.MatchString(t) {
		return t
	}
	// Reserve some static space for common string lengths
	cleaned_space := [40]byte{}
	cleaned := cleaned_space[:0]
	for _, c := range t {
		if !(c >= '0' && c <= '9') {
			// We know that 'c' is a single-byte character, so this type cast is safe
			cleaned = append(cleaned, byte(c))
		}
	}
	return string(cleaned)
}
*/

func numbersOnly(t *string) {
	for _, c := range *t {
		if !(c >= '0') || !(c <= '9') {
			*t = strings.Replace(*t, string(c), " ", -1)
		}
	}
}

// addCountryCode will find the first possible valid number for the
// given set of country codes.  This function may be a bottleneck
// for large amounts of mobile numbers. More prevalent country codes
// must be placed at the top of the country code slice to improve
// performance.
func addCountryCode(t *string, cs []string) {
	mn, _ := libphonenumber.Parse(*t, cs[0])
	if libphonenumber.IsValidNumberForRegion(mn, cs[0]) {
		*t = fmt.Sprintf("%v%v", *mn.CountryCode, *mn.NationalNumber)
		return
	}
	if len(cs) > 1 {
		addCountryCode(t, cs[1:])
	} else {
		*t = ""
	}
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
