package orders

import "regexp"

func IsNumeric(word string) bool {
	return regexp.MustCompile(`\d`).MatchString(word)
}
