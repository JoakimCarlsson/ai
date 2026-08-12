package main

import (
	"context"
	"fmt"
	"maps"
	"sort"
)

// kind identifies which catalog a model belongs in. One provider endpoint can
// return several kinds, and one provider can expose one kind through several
// endpoints, so routing is by kind rather than by request.
type kind string

const (
	kindChat          kind = "chat"
	kindImage         kind = "image"
	kindSpeech        kind = "speech"
	kindTranscription kind = "transcription"
	kindEmbedding     kind = "embedding"
	kindRerank        kind = "rerank"
)

// model is a provider model normalized into the Go literals a catalog entry is
// written with. Only fields the API actually describes are set; everything else
// is carried over from the existing catalog or filled from target defaults.
type model struct {
	kind     kind
	apiModel string
	fields   map[string]string
	// seed holds fields used only when the model is new to the catalog, for
	// values a provider publishes too poorly to overwrite a curated one with.
	seed map[string]string
}

// target is one generated models.go file.
type target struct {
	kind       kind
	path       string
	pkg        string
	importPath string
	typeExpr   string
	source     string
	// idFull keeps the provider's whole model ID in the catalog ID instead of
	// dropping the vendor prefix.
	idFull bool
	// keepStale leaves entries the source stopped listing in place instead of
	// deleting them, for sources that index a provider rather than serve it.
	keepStale bool
	doc       []string
	order     []string
	defaults  map[string]string
	idPrefix  string
}

type provider struct {
	name string
	// source names where this provider's data comes from, so a run report says
	// which catalogs were written from which source.
	source  string
	fetch   func(ctx context.Context) ([]model, error)
	targets []target
}

// matchExisting pairs fetched models with the catalog entries they update.
// Exact API model matches are paired first; a model the catalog only holds
// under a dated snapshot slug is paired second, so a provider dropping the date
// from a slug updates the entry rather than adding a second one beside it. Each
// entry is claimed once, and seen records which catalog entries survived.
func matchExisting(
	fetched []model,
	cat *catalog,
	seen map[string]bool,
) map[string]*entry {
	matched := make(map[string]*entry, len(fetched))

	for _, m := range fetched {
		if e, ok := cat.entries[m.apiModel]; ok {
			matched[m.apiModel] = e
			seen[m.apiModel] = true
		}
	}

	undatedEntries := make(map[string]*entry, len(cat.entries))
	ambiguous := make(map[string]bool)
	for api, e := range cat.entries {
		key := undated(api)
		if key == api || seen[api] {
			continue
		}
		if _, ok := undatedEntries[key]; ok {
			ambiguous[key] = true
		}
		undatedEntries[key] = e
	}

	for _, m := range fetched {
		if matched[m.apiModel] != nil {
			continue
		}
		key := undated(m.apiModel)
		e := undatedEntries[key]
		if e == nil || ambiguous[key] || seen[e.apiModel] {
			continue
		}
		matched[m.apiModel] = e
		seen[e.apiModel] = true
	}

	return matched
}

// checkUnique guards the two invariants a catalog file must hold: no duplicate
// constant name, and no duplicate ID value in the Models map.
func checkUnique(entries []*entry) error {
	names := make(map[string]bool, len(entries))
	ids := make(map[string]string, len(entries))

	for _, e := range entries {
		if names[e.constName] {
			return fmt.Errorf("duplicate constant %s", e.constName)
		}
		names[e.constName] = true

		if held, ok := ids[e.constVal]; ok {
			return fmt.Errorf(
				"constants %s and %s share the ID %q",
				held,
				e.constName,
				e.constVal,
			)
		}
		ids[e.constVal] = e.constName
	}
	return nil
}

// result is what a single target sync did, for the run report.
type result struct {
	target  target
	added   []string
	updated int
	removed []string
	stale   []string
}

// syncTarget merges the fetched models into the existing catalog and returns
// the file to write. Existing entries keep their constant name, their ID and
// every field the API does not describe. Models the source no longer returns
// are dropped, unless the target keeps stale entries, in which case they are
// left in place and reported.
func syncTarget(
	t target,
	fetched []model,
	cat *catalog,
	date string,
) (string, result, error) {
	res := result{target: t}
	taken := maps.Clone(cat.names)
	takenIDs := make(map[string]bool, len(cat.entries))
	for _, e := range cat.entries {
		takenIDs[e.constVal] = true
	}

	sort.Slice(fetched, func(i, j int) bool {
		return fetched[i].apiModel < fetched[j].apiModel
	})

	seen := make(map[string]bool, len(fetched))
	out := make([]*entry, 0, len(fetched))
	matched := matchExisting(fetched, cat, seen)

	for _, m := range fetched {
		existing := matched[m.apiModel]

		e := &entry{apiModel: m.apiModel, fields: make(map[string]string)}
		if existing != nil {
			e.constName = existing.constName
			e.constVal = existing.constVal
			maps.Copy(e.fields, existing.fields)
			res.updated++
		} else {
			e.constName = uniqueConstName(m.apiModel, taken)
			taken[e.constName] = true
			e.constVal = uniqueID(m.apiModel, t.idPrefix, t.idFull, takenIDs)
			takenIDs[e.constVal] = true
			maps.Copy(e.fields, t.defaults)
			maps.Copy(e.fields, m.seed)
			res.added = append(res.added, m.apiModel)
		}

		maps.Copy(e.fields, m.fields)
		e.fields["ID"] = e.constName

		out = append(out, e)
	}

	for api, e := range cat.entries {
		if seen[api] {
			continue
		}
		label := e.constName + " (" + api + ")"
		if t.keepStale {
			res.stale = append(res.stale, label)
			out = append(out, e)
			continue
		}
		res.removed = append(res.removed, label)
	}
	sort.Strings(res.removed)
	sort.Strings(res.stale)
	sort.Slice(out, func(i, j int) bool {
		return out[i].apiModel < out[j].apiModel
	})

	if err := checkUnique(out); err != nil {
		return "", res, fmt.Errorf("%s: %w", t.path, err)
	}

	src, err := render(t, out, date)
	if err != nil {
		return "", res, err
	}
	return src, res, nil
}
