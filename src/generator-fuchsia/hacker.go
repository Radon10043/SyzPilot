package main

import (
	"fmt"
	"regexp"

	"github.com/Radon10043/cloud/src/pkg/pool"
)

type Hacker struct {
	HackPatterns   map[string]func(string) string
	UnhackPatterns map[string]func(string) string
}

// Hack modifies syzlang spec via HackPatterns, returns modified spec
func (h *Hacker) Hack(syzl string) string {
	newsyzl := syzl
	for k, f := range h.HackPatterns {
		re := regexp.MustCompile(k)
		newsyzl = re.ReplaceAllStringFunc(newsyzl, func(match string) string {
			replacement := f(match)
			submatches := re.FindStringSubmatchIndex(match)
			return string(re.ExpandString(nil, replacement, match, submatches))
		})
	}
	return newsyzl
}

// Unhack modifies syzlang spec via UnhackPatterns, returns modified spec
func (h *Hacker) Unhack(syzl string) string {
	newsyzl := syzl
	for k, f := range h.UnhackPatterns {
		re := regexp.MustCompile(k)
		newsyzl = re.ReplaceAllStringFunc(newsyzl, func(match string) string {
			replacement := f(match)
			submatches := re.FindStringSubmatchIndex(match)
			return string(re.ExpandString(nil, replacement, match, submatches))
		})
	}
	return newsyzl
}

// HackSpecPool modifies SpecPool via HackPatterns, return modified pool
func (h *Hacker) HackSpecPool(po *pool.SpecPool) *pool.SpecPool {
	for _, v := range *po {
		orig := v.Code
		v.Code = h.Hack(v.Code)
		if v.Code != orig {
			v.Valid = false
		}
	}
	return po
}

// UnhackSpecPool modifies SpecPool via UnhackPatterns, return modified pool
func (h *Hacker) UnhackSpecPool(po *pool.SpecPool) *pool.SpecPool {
	for _, v := range *po {
		orig := v.Code
		v.Code = h.Unhack(v.Code)
		if v.Code != orig {
			v.Valid = false
		}
	}
	return po
}

// Deduplicate deduplicates redeclared nodes in the syzlang spec, returns deduplicated spec
// and error messages
func (h *Hacker) Deduplicate(syzl string) (string, error) {
	po, err := pool.NewSpecPoolFromSyzlang(syzl)
	if err != nil {
		return "", fmt.Errorf("failed to convert syzlang spec to pool during deduplication: %v\n", err)
	}
	return po.Syzlang(), nil
}

type hackerOptions func(*Hacker)

// WithHackPatterns sets HackPatterns field of Hacker
func WithHackPatterns(patterns map[string]func(string) string) hackerOptions {
	return func(h *Hacker) {
		h.HackPatterns = patterns
	}
}

// WithUnhackPatterns sets UnhackPatterns field of Hacker
func WithUnhackPatterns(patterns map[string]func(string) string) hackerOptions {
	return func(h *Hacker) {
		h.UnhackPatterns = patterns
	}
}

// NewHakcer returns a Hacker struct with user provided options
func NewHacker(opts ...hackerOptions) Hacker {
	h := Hacker{}
	for _, opt := range opts {
		opt(&h)
	}
	return h
}
