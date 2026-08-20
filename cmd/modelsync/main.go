// Package main regenerates the provider model catalogs from the providers'
// own model APIs.
//
// It takes no arguments: every registered provider is fetched, every catalog it
// owns is rewritten as a full mirror of what the API returns, and a report of
// what was added, updated and removed is printed.
//
//	go run ./cmd/modelsync
//
// Entries already in a catalog keep their Go constant name, their ID and every
// field the API does not describe, so regenerating never renames an exported
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	date := time.Now().UTC().Format("2006-01-02")
	providers := []provider{openRouter(), berget(), modelsDev()}
	if err := checkTargets(providers); err != nil {
		return err
	}

	var failures []error
	for _, p := range providers {
		if err := syncProvider(ctx, root, p, date); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", p.name, err))
			fmt.Printf("%s: FAILED: %v\n", p.name, err)
		}
	}
	return errors.Join(failures...)
}

func syncProvider(
	ctx context.Context,
	root string,
	p provider,
	date string,
) error {
	fmt.Printf("\n== %s (%s)\n", p.name, p.source)

	models, err := p.fetch(ctx)
	if err != nil {
		return err
	}

	for _, t := range p.targets {
		var forTarget []model
		for _, m := range models {
			if m.kind == t.kind {
				forTarget = append(forTarget, m)
			}
		}
		if len(forTarget) == 0 {
			fmt.Printf("%s: no models returned, left untouched\n", t.path)
			continue
		}

		path := filepath.Join(root, filepath.FromSlash(t.path))
		cat, err := readCatalog(path)
		if err != nil {
			return err
		}

		src, res, err := syncTarget(t, forTarget, cat, date)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			return err
		}
		report(res)
	}
	return nil
}

// checkTargets rejects a registry where two providers write the same catalog.
// Precedence between sources is a decision to make when registering a provider,
// not something to discover from whichever one happened to run last.
func checkTargets(providers []provider) error {
	owner := make(map[string]string)
	for _, p := range providers {
		for _, t := range p.targets {
			if held, ok := owner[t.path]; ok {
				return fmt.Errorf(
					"%s is claimed by both %s and %s",
					t.path,
					held,
					p.name,
				)
			}
			owner[t.path] = p.name
		}
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
