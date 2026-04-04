package utils

import "regexp"

func CheckLuhn(number string) bool {
	matched, _ := regexp.MatchString(`^\d+$`, number)
	if !matched {
		return false
	}
	sum := 0
	alt := false

	for i := len(number) - 1; i >= 0; i-- {
		n := int(number[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}
