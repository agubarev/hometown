package util

import "flag"

func IsTestMode() bool {
	return flag.Lookup("test.v") != nil
}
