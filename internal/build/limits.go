package build

import "time"

type Limits struct {
	MaxExtractedBytes int64
	MaxFiles          int
	MaxBuildTime      time.Duration
	MaxOutputBytes    int64
}

func DefaultLimits() Limits {
	return Limits{
		MaxExtractedBytes: 50 << 20,
		MaxFiles:          5000,
		MaxBuildTime:      2 * time.Minute,
		MaxOutputBytes:    100 << 20,
	}
}
