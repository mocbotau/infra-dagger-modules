package main

import "fmt"

var ErrVersionBumpSkipped = fmt.Errorf("version bump skipped due to [skip] marker in commit message")
