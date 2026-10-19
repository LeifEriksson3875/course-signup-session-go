package main

import (
	"testing"
	"time"
)

func TestCourseDeadline(t *testing.T) {
	tests := []struct {
		name     string
		now, due time.Time
		want     bool
	}{{"before", time.Unix(10, 0), time.Unix(20, 0), true}, {"after", time.Unix(30, 0), time.Unix(20, 0), false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.now.Before(tt.due)
			if got != tt.want {
				t.Fatalf("deadline decision=%v want %v", got, tt.want)
			}
		})
	}
}
