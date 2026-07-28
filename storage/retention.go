package storage

import "time"

func isExpired(modifiedAt time.Time, expiryDays int, now time.Time) bool {
	if expiryDays <= 0 {
		return false
	}

	return now.Sub(modifiedAt).Hours()/24 > float64(expiryDays)
}
