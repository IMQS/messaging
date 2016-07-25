package messaging

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ttacon/libphonenumber"
)

// cleanMSISDNs receives a list of MSISDNs and runs a series
// of checks to ensure that they are valid mobile numbers for the countries
// provided. Invalid and duplicate numbers are ignored and removed from the reply.
func cleanMSISDNs(ns, cs []string) []string {
	pNs := []string{}
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
