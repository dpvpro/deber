// Package util includes useful helper functions
package util

import (
	"slices"
	"github.com/docker/docker/api/types/mount"
)

// CompareMounts function simply compares if given mounts are equal
func CompareMounts(a, b []mount.Mount) bool {
	if len(a) != len(b) {
		return false
	}

	matches := 0
	for _, aMount := range a {
	   if slices.Contains(b, aMount) {
			matches++
		}
	}
	
	return matches == len(a) 
}
