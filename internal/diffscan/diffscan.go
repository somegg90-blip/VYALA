package diffscan

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

func ChangedFiles(repoRoot, baseRef string) ([]string, error) {
	out, err := runGit(repoRoot, "diff", "--name-only", "--diff-filter=ACMR", baseRef+"...HEAD")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func AddedLines(repoRoot, baseRef, file string) (map[int]bool, error) {
	out, err := runGit(repoRoot, "diff", "--unified=0", baseRef+"...HEAD", "--", file)
	if err != nil {
		return nil, err
	}
	lines := map[int]bool{}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		m := hunkHeaderRe.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		start, _ := strconv.Atoi(m[1])
		count := 1
		if m[2] != "" {
			count, _ = strconv.Atoi(m[2])
		}
		if count == 0 {
			continue
		}
		for i := 0; i < count; i++ {
			lines[start+i] = true
		}
	}
	return lines, scanner.Err()
}

func runGit(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w (stderr: %s)", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}