package messaging

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// cleanMSISDNs receives a list of MSISDNs and runs a series
// of checks to ensure that they are valid mobile numbers for the countries
// provided. Invalid and duplicate numbers are ignored and removed from the reply.
func cleanMSISDNs(ns, cs []string) []string {
	// lenNs := len(ns)
	pNs := []string{}
	workers := make(chan string)
	numbers := make(chan string)

	// spawn four worker goroutines
	var wg sync.WaitGroup

	go func() {
		n := <-numbers
		if n != "" {
			pNs = append(pNs, n)
		}
	}()

	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			for gn := range workers {
				// only allow numbers
				numbersOnly(&gn)

				// remove all spaces
				gn = strings.Replace(gn, " ", "", -1)

				// add the country code and verify if number is valid
				addCountryCode(&gn)
				numbers <- gn
			}
			wg.Done()
		}()
	}

	t1 := time.Now()
	for _, n := range ns {
		workers <- n
	}

	go func() {
		wg.Wait()
		close(workers)
	}()

	pNs = removeDuplicates(pNs)
	t4 := time.Now()
	fmt.Println("Dups: ", t4.Sub(t1))
	// close(numbers)
	return ns
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
// func addCountryCode(t *string, cs []string) {
// 	mn, _ := libphonenumber.Parse(*t, cs[0])
// 	if libphonenumber.IsValidNumberForRegion(mn, cs[0]) {
// 		*t = fmt.Sprintf("%v%v", *mn.CountryCode, *mn.NationalNumber)
// 		return
// 	}
// 	if len(cs) > 1 {
// 		addCountryCode(t, cs[1:])
// 	} else {
// 		*t = ""
// 	}
// }

func addCountryCode(t *string) {
	switch {
	case strings.HasPrefix(*t, "0") && len(*t) == 10:
		*t = "27" + (*t)[1:len(*t)]
	case strings.HasPrefix(*t, "27") && len(*t) == 11:
		break
	case len(*t) == 9:
		*t = "27" + *t
	default:
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
