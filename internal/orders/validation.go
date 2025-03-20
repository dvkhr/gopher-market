package orders

import "regexp"

func Is_numeric(word string) bool {
	return regexp.MustCompile(`\d`).MatchString(word)
}
