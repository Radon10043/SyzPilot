// Copyright 2019 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fidlgen

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Formatter formats generated source code
type Formatter interface {
	// Format formats source code.
	Format(source []byte) ([]byte, error)
}

// identifyFormatter returns the input unmodified
type identityFormatter struct{}

func (f identityFormatter) Format(source []byte) ([]byte, error) {
	return source, nil
}

// externalFormatter formats a writer stream.
type externalFormatter struct {
	path  string
	args  []string
	limit int
}

var _ = []Formatter{identityFormatter{}, externalFormatter{}}

// NewFormatter creates a new external formatter.
//
// The `path` needs to either
// * Point to an executable which formats stdin and outputs it to stdout;
// * An empty string, in which case no formatting will occur.
func NewFormatter(path string, args ...string) Formatter {
	if path == "" {
		return identityFormatter{}
	}
	return externalFormatter{
		path: path,
		args: args,
	}
}

// NewFormatterWithSizeLimit creates a new external formatter that doesn't
// attempt to format sources over a specified size.
//
// If the source is within the size limit, but the formatted output exceeds it,
// then it discards the output and returns the unformatted source. This ensures
// that running it multiple times (which happens when we reformat golden files)
// will either invoke the external formatter every time or not at all.
//
// The `path` needs to either * Point to an executable which formats stdin and
// outputs it to stdout; * An empty string, in which case no formatting will
// occur.
func NewFormatterWithSizeLimit(limit int, path string, args ...string) Formatter {
	if path == "" {
		return identityFormatter{}
	}
	return externalFormatter{
		path:  path,
		args:  args,
		limit: limit,
	}
}

func (f externalFormatter) Format(source []byte) ([]byte, error) {
	if f.limit > 0 && len(source) > f.limit {
		return source, nil
	}
	cmd := exec.Command(f.path, f.args...)
	formattedBuf := new(bytes.Buffer)
	cmd.Stdout = formattedBuf
	errBuf := new(bytes.Buffer)
	cmd.Stderr = errBuf
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("Error creating stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Error starting formatter process: %w", err)
	}
	start := time.Now()
	if _, err := in.Write(source); err != nil {
		return nil, fmt.Errorf("Error writing stdin: %w", err)
	}
	if err := in.Close(); err != nil {
		return nil, fmt.Errorf("Error closing stdin: %w", err)
	}
	err = cmd.Wait()
	end := time.Now()
	// On builders, formatting can take a long time, and it's nice to know when this happens.
	// We'll emit a warning describing what we're trying to do to help diagnose slow downs if
	// they happen.
	formatting_duration := end.Sub(start)
	if formatting_duration > 2*time.Minute {
		fmt.Fprintf(os.Stderr, "WARNING: Slow formatting operation took %v: %v %v\n",
			formatting_duration,
			f.path,
			strings.Join(f.args, " "))
	}

	if err != nil {
		if errContent := errBuf.Bytes(); len(errContent) != 0 {
			return nil, fmt.Errorf("Formatter (%v) error: %w (stderr: %s)", cmd, err, string(errContent))
		}
		return nil, fmt.Errorf("Formatter (%v) error but stderr was empty: %w", cmd, err)
	}
	if f.limit > 0 && formattedBuf.Len() > f.limit {
		return source, nil
	}
	return formattedBuf.Bytes(), nil
}
