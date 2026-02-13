package ports

import "time"

// Clock provides current time.
type Clock func() time.Time
