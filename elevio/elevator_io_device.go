package elevio

import (
	"strconv"
	"strings"
)

func StringToCabArray(input string) [N_Floors]bool {
	var cabArray [N_Floors]bool
	parts := strings.Split(input, ",")
	for i, part := range parts {
		if part == "true" {
			cabArray[i] = true
		} else {
			cabArray[i] = false
		}
	}
	return cabArray
}

func CabArrayToString(input [N_Floors]bool) string {
	var strArr []string
	for _, b := range input {
		strArr = append(strArr, strconv.FormatBool(b))
	}
	return strings.Join(strArr, ",")
}
