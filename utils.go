package messaging

import "strings"

// NormalizeMSISDNs receives a list of MSISDNs and runs a series
// of checks to ensure that they are valid South African mobile numbers.
// Invalid and duplicate numbers are ignored and removed from the reply.
// Future enhancements could support multiple countries and also
// consider certain network codes (e.g. 2783 -> MTN) for validation.
func NormalizeMSISDNs(ns []string) []string {
	for i := len(ns) - 1; i >= 0; i-- {
		n := ns[i]
		// only allow numbers
		numbersOnly(&n)
		// remove all spaces
		n = strings.Replace(n, " ", "", -1)
		// add 27 for South African MSISDNs
		addCountryCode(&n)
		if n == "" {
			ns = append(ns[:i], ns[i+1:]...)
		} else {
			ns[i] = n
		}
	}
	ns = removeDuplicates(ns)
	return ns
}

func numbersOnly(t *string) {
	for _, c := range *t {
		if !(c >= '0') || !(c <= '9') {
			*t = strings.Replace(*t, string(c), " ", -1)
		}
	}
}

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
