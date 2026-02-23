package scraper

import (
	"strings"
)

func mergeUnique(existing, page []Company, seen map[string]struct{}) []Company {
	// for every item in page you check the seen list.
	// if the scene list has something carry on
	// if not add to list
	for _, company := range page {
		// Skip if empty strings
		if strings.TrimSpace(company.Name) == "" {
			continue
		}
		company.Name = strings.ToLower(company.Name)
		if _, ok := seen[company.Name]; ok {
			continue
		}
		seen[company.Name] = struct{}{}
		existing = append(existing, company)
	}

	return existing
}
