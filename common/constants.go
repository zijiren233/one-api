package common

import "time"

var (
	StartTime = time.Now().Unix() // unit: second
	Version   = "v0.0.0"          // this hard coding will be replaced automatically when building, no need to manually change
)
