package mcp

import (
	"strings"
	"testing"
)

func TestIsBlacklistedFilesystemPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{path: "/repo/ui/node_modules/react/index.js", want: true},
		{path: "/repo/rust/target/debug/app", want: true},
		{path: "/repo/.git/config", want: true},
		{path: "/repo/ui/src/App.tsx", want: false},
	}

	for _, tc := range cases {
		if got := isBlacklistedFilesystemPath(tc.path, nil); got != tc.want {
			t.Fatalf("isBlacklistedFilesystemPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestValidateFilesystemToolArgs(t *testing.T) {
	err := validateFilesystemToolArgs("mcp_filesystem_read_file", map[string]interface{}{
		"path": "/repo/ui/node_modules/react/index.js",
	}, nil)
	if err == nil {
		t.Fatal("expected blacklisted path to be rejected")
	}

	err = validateFilesystemToolArgs("mcp_filesystem_read_file", map[string]interface{}{
		"path": "/repo/ui/src/App.tsx",
	}, nil)
	if err != nil {
		t.Fatalf("expected normal path to pass, got %v", err)
	}
}

func TestFilterFilesystemToolResult(t *testing.T) {
	result := &ToolResult{
		Success: true,
		Data: "Directory listing for: /repo/ui\n\n" +
			"[DIR]  node_modules (file:///repo/ui/node_modules)\n" +
			"[DIR]  src (file:///repo/ui/src)\n" +
			"[FILE] package.json (file:///repo/ui/package.json) - 100 bytes\n",
	}

	filtered := filterFilesystemToolResult("mcp_filesystem_list_directory", result, nil)
	text, _ := filtered.Data.(string)
	if text == "" {
		t.Fatal("expected filtered result text")
	}
	if wantGone := "node_modules"; strings.Contains(text, wantGone) {
		t.Fatalf("expected blacklisted entry %q to be removed: %s", wantGone, text)
	}
	if !strings.Contains(text, "src") {
		t.Fatalf("expected non-blacklisted entry to remain: %s", text)
	}
}
