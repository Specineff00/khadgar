package scraper

import (
	"strings"

	"khadgar"
)

func toCompany(in khadgar.PersonalisedCompaniesPersonalisedCompaniesCompany) Company {
	return Company{
		Name:             strings.TrimSpace(in.Name),
		ShortDescription: strings.TrimSpace(in.ShortDescription),
		Size:             strings.TrimSpace(in.Size.Value),
	}
}

func toCompanies(pcs []khadgar.PersonalisedCompaniesPersonalisedCompaniesCompany) []Company {
	out := make([]Company, 0, len(pcs))
	for _, pc := range pcs {
		out = append(out, toCompany(pc))
	}
	return out
}
