package archiver

import "testing"

func TestGetBackupPath(t *testing.T) {
	cases := []struct {
		dst      string
		dstMD5   string
		expected string
	}{
		{
			dst:      "/tmp/agent/workflow.pkl",
			dstMD5:   "abcd1234",
			expected: "/tmp/agent/workflow_abcd1234.pkl",
		},
		{
			dst:      "data.txt",
			dstMD5:   "deadbeef",
			expected: "data_deadbeef.txt",
		},
	}

	for _, c := range cases {
		got := getBackupPath(c.dst, c.dstMD5)
		if got != c.expected {
			t.Errorf("getBackupPath(%q, %q) = %q, want %q", c.dst, c.dstMD5, got, c.expected)
		}
	}
}
