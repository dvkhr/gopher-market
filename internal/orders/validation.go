package orders

import "regexp"

var numericRegex = regexp.MustCompile(`^[0-9]+$`)

func IsNumeric(word string) bool {
	return numericRegex.MatchString(word)
}
