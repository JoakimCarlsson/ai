package main

import (
	"context"
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
	// idVerbatim keeps the provider's model ID as the catalog ID instead of
	// deriving a shortened one.
	idVerbatim bool
	doc        []string
	order      []string
	defaults   map[string]string
	idPrefix   string
}

type provider struct {
	name    string
	fetch   func(ctx context.Context) ([]model, error)
	targets []target
}

// result is what a single target sync did, for the run report.
type result struct {
	target  target
	added   []string
	updated int
	removed []string
}

// syncTarget merges the fetched models into the existing catalog and returns
// the file to write. Existing entries keep their constant name, their ID and
// every field the API does not describe; models the API no longer returns are
// dropped.
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

	for _, m := range fetched {
		seen[m.apiModel] = true
		existing := cat.entries[m.apiModel]

		e := &entry{fields: make(map[string]string)}
		if existing != nil {
			e.constName = existing.constName
			e.constVal = existing.constVal
			maps.Copy(e.fields, existing.fields)
			res.updated++
		} else {
			e.constName = uniqueConstName(m.apiModel, taken)
			taken[e.constName] = true
			if t.idVerbatim {
				e.constVal = m.apiModel
			} else {
				e.constVal = uniqueID(m.apiModel, t.idPrefix, takenIDs)
			}
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
		if !seen[api] {
			res.removed = append(res.removed, e.constName+" ("+api+")")
		}
	}
	sort.Strings(res.removed)

	src, err := render(t, out, date)
	if err != nil {
		return "", res, err
	}
	return src, res, nil
}
