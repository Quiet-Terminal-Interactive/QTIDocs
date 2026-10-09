package build

import "testing"

func TestDefaultLimits(t *testing.T) {
	l := DefaultLimits()
	if l.MaxExtractedBytes <= 0 {
		t.Error("MaxExtractedBytes should be positive")
	}
	if l.MaxFiles <= 0 {
		t.Error("MaxFiles should be positive")
	}
	if l.MaxBuildTime <= 0 {
		t.Error("MaxBuildTime should be positive")
	}
	if l.MaxOutputBytes <= 0 {
		t.Error("MaxOutputBytes should be positive")
	}
}
