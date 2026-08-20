// Package main regenerates the provider model catalogs from model-sync.
//
// It takes no arguments: https://github.com/JoakimCarlsson/model-sync publishes
// one api.json describing every provider and every model it serves, that
// document is fetched once, every catalog it owns is rewritten as a full mirror
// of the slice of it that catalog covers, and a report of what was added,
// updated and removed is printed.
//
//	go run ./cmd/modelsync
//
// Entries already in a catalog keep their Go constant name, their ID and every
// field the source does not describe, so regenerating never renames an exported
// identifier or drops hand-recorded detail.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "modelsync:", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	if err := checkTargets(targets); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Printf("fetching %s\n", sourceURL)
	index, err := fetchIndex(ctx, sourceURL)
	if err != nil {
		return err
	}

	date := time.Now().UTC().Format("2006-01-02")

	var failures []error
	for _, t := range targets {
		if err := syncCatalog(root, t, index, date); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", t.path, err))
			fmt.Printf("%s: FAILED: %v\n", t.path, err)
		}
	}
	return errors.Join(failures...)
}

func syncCatalog(
	root string,
	t target,
	index *apiIndex,
	date string,
) error {
	sourced := index.models(t.source, string(t.kind))
	if len(sourced) == 0 {
		fmt.Printf(
			"%s: no %s models listed for %s, left untouched\n",
			t.path,
			t.kind,
			t.source,
		)
		return nil
	}

	fetched, duplicates := dedupe(t, sourced)
	if duplicates > 0 {
		fmt.Printf(
			"%s: source repeats %d model ID(s), first listing kept\n",
			t.path,
			duplicates,
		)
	}

	path := filepath.Join(root, filepath.FromSlash(t.path))
	cat, err := readCatalog(path)
	if err != nil {
		return err
	}

	src, res, err := syncTarget(t, fetched, cat, date)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		return err
	}
	report(res)
	return nil
}

// dedupe drops repeats of a model ID the source lists more than once, keeping
// the first. A catalog holds one entry per API model, since that is the key an
// entry is matched by, so a second entry under the same ID would displace the
// first the next time the catalog is read.
func dedupe(t target, sourced []apiModel) ([]model, int) {
	fetched := make([]model, 0, len(sourced))
	seen := make(map[string]bool, len(sourced))

	duplicates := 0
	for _, m := range sourced {
		id := m.apiID()
		if seen[id] {
			duplicates++
			continue
		}
		seen[id] = true
		fetched = append(fetched, modelFor(t, m))
	}
	return fetched, duplicates
}

// checkTargets rejects a registry where two entries write the same catalog.
// Which slice of the source a catalog mirrors is a decision to make when
// registering it, not something to discover from whichever entry happened to
// run last.
func checkTargets(targets []target) error {
	owner := make(map[string]string)
	for _, t := range targets {
		claim := t.source + "/" + string(t.kind)
		if held, ok := owner[t.path]; ok {
			return fmt.Errorf(
				"%s is claimed by both %s and %s",
				t.path,
				held,
				claim,
			)
		}
		owner[t.path] = claim
	}
	return nil
}

func report(res result) {
	fmt.Printf(
		"%s: %d updated, %d added, %d removed\n",
		res.target.path,
		res.updated,
		len(res.added),
		len(res.removed),
	)
	for _, a := range res.added {
		fmt.Printf("  + %s\n", a)
	}
	for _, r := range res.removed {
		fmt.Printf("  - %s\n", r)
	}
}

// repoRoot walks up from the working directory to the module workspace, so the
// tool can be run from anywhere in the repository.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(
			filepath.Join(dir, "go.work.example"),
		); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("repository root not found")
		}
		dir = parent
	}
}
