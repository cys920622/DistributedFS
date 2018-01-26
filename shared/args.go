package shared

import "time"

// todo - remove
type Args struct {
	Filename, B string
}

type ClientRegistrationInfo struct {
	ClientId int
	ClientAddress string
	LatestHeartbeat time.Time
}

type ClientHeartbeat struct {
	ClientId int
	Timestamp time.Time
}