package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type NavPage struct {
	Title   string
	Path    string
	Section string
	Depth   int
}

type mkdocsConfig struct {
	SiteName string `yaml:"site_name"`
	SiteURL  string `yaml:"site_url"`
	Nav      []any  `yaml:"nav"`
}

func parseConfig(
	path string,
) (siteName, siteURL string, pages []NavPage, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", nil, fmt.Errorf(
			"reading config: %w",
			err,
		)
	}

	var cfg mkdocsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", "", nil, fmt.Errorf(
			"parsing config: %w",
			err,
		)
	}

	pages = walkNav(cfg.Nav, "", 0)
	return cfg.SiteName, cfg.SiteURL, pages, nil
}

func walkNav(items []any, section string, depth int) []NavPage {
	var pages []NavPage

	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		for title, val := range m {
			switch v := val.(type) {
			case string:
				pages = append(pages, NavPage{
					Title:   title,
					Path:    v,
					Section: section,
					Depth:   depth,
				})
			case []any:
				pages = append(
					pages,
					walkNav(v, title, depth+1)...,
				)
			}
		}
	}

	return pages
}
