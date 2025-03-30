// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package memory_test

import (
	"runtime"
	"testing"

	"github.com/tetsuo/fortune/internal/memory"
)

func TestRead(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Log("skipping memory read tests, not linux")
		return
	}
	_, err := memory.ReadSystemStats()
	if err != nil {
		t.Fatal(err)
	}
	_, err = memory.ReadProcessStats()
	if err != nil {
		t.Fatal(err)
	}

	// We can't really test ReadCgroupStats, because we may or may not be in a cgroup.
}

func TestFormat(t *testing.T) {
	for _, test := range []struct {
		m    uint64
		want string
	}{
		{0, "0 B"},
		{1022, "1022 B"},
		{2500, "2.44 K"},
		{4096, "4.00 K"},
		{2_000_000, "1.91 M"},
		{18_000_000_000, "16.76 G"},
	} {
		got := memory.Format(test.m)
		if got != test.want {
			t.Errorf("%d: got %q, want %q", test.m, got, test.want)
		}
	}
}
