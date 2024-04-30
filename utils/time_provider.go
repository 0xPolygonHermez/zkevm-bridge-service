package utils

import (
	"time"
)

// TimeProvider is a interface for classes that needs time and we want to be able to unittest it
type TimeProvider interface {
	// Now returns current time
	Now() time.Time
}

// TimeProviderSystemLocalTime is the default implementation of TimeProvider
type TimeProviderSystemLocalTime struct{}

func NewTimeProviderSystemLocalTime() *TimeProviderSystemLocalTime {
	return &TimeProviderSystemLocalTime{}
}

// Now returns current time
func (d TimeProviderSystemLocalTime) Now() time.Time {
	return time.Now()
}

// TimeProviderFixedTime is a implementation that always returns the same time
// that is useful for testing
type TimeProviderFixedTime struct {
	FixedTime time.Time
}

// Now returns current time
func (d TimeProviderFixedTime) Now() time.Time {
	return d.FixedTime
}
