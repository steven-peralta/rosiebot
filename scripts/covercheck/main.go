package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type threshold struct {
	prefix string
	min    float64
}

type thresholds []threshold

func (t *thresholds) String() string { return fmt.Sprint(*t) }

func (t *thresholds) Set(v string) error {
	prefix, minStr, ok := strings.Cut(v, "=")
	if !ok {
		return fmt.Errorf("expected prefix=percent, got %q", v)
	}
	minPct, err := strconv.ParseFloat(minStr, 64)
	if err != nil {
		return fmt.Errorf("bad percent in %q: %w", v, err)
	}
	*t = append(*t, threshold{prefix: prefix, min: minPct})
	return nil
}

type tally struct {
	statements int
	covered    int
}

func (t tally) percent() float64 {
	if t.statements == 0 {
		return 0
	}
	return 100 * float64(t.covered) / float64(t.statements)
}

func main() {
	var (
		profile = flag.String("profile", "coverage.out", "coverage profile produced by go test")
		module  = flag.String("module", "github.com/steven-peralta/rosiebot", "module path prefix")
		exclude = flag.String("exclude", `/(gen|mocks|cmd|scripts)/`, "regexp of file paths to ignore")
		overall = flag.Float64("overall", 0, "minimum overall percent across included files")
		mins    thresholds
	)
	flag.Var(&mins, "min", "per-package minimum as prefix=percent, repeatable")
	flag.Parse()

	files, err := parseProfile(*profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	skip := regexp.MustCompile(*exclude)
	byPkg := map[string]tally{}
	var total tally
	for file, t := range files {
		if skip.MatchString(file) {
			continue
		}
		pkg := strings.TrimPrefix(path.Dir(file), *module+"/")
		acc := byPkg[pkg]
		acc.statements += t.statements
		acc.covered += t.covered
		byPkg[pkg] = acc
		total.statements += t.statements
		total.covered += t.covered
	}

	pkgs := make([]string, 0, len(byPkg))
	for pkg := range byPkg {
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)
	for _, pkg := range pkgs {
		fmt.Printf("%-45s %6.1f%%\n", pkg, byPkg[pkg].percent())
	}

	failed := false
	for _, th := range mins {
		var acc tally
		for pkg, t := range byPkg {
			if pkg == th.prefix || strings.HasPrefix(pkg, th.prefix+"/") {
				acc.statements += t.statements
				acc.covered += t.covered
			}
		}
		if acc.statements == 0 {
			fmt.Printf("%-45s   skip  (no statements yet, min %.0f%%)\n", th.prefix, th.min)
			continue
		}
		status := "ok"
		if acc.percent() < th.min {
			status = "FAIL"
			failed = true
		}
		fmt.Printf("%-45s %6.1f%%  %s (min %.0f%%)\n", th.prefix, acc.percent(), status, th.min)
	}

	if total.statements > 0 || *overall > 0 {
		status := "ok"
		if total.statements > 0 && total.percent() < *overall {
			status = "FAIL"
			failed = true
		}
		fmt.Printf("%-45s %6.1f%%  %s (min %.0f%%)\n", "overall", total.percent(), status, *overall)
	}

	if failed {
		os.Exit(1)
	}
}

func parseProfile(name string) (map[string]tally, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}

	files := map[string]tally{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") || line == "" {
			continue
		}
		file, rest, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("malformed profile line: %q", line)
		}
		fields := strings.Fields(rest)
		if len(fields) != 3 {
			return nil, fmt.Errorf("malformed profile line: %q", line)
		}
		statements, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("malformed statement count in %q: %w", line, err)
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("malformed hit count in %q: %w", line, err)
		}
		t := files[file]
		t.statements += statements
		if count > 0 {
			t.covered += statements
		}
		files[file] = t
	}
	return files, scanner.Err()
}
