package scraper

func shouldStop(pageSize, limit int) bool {
	if limit <= 0 {
		return true
	}
	return pageSize < limit
}

func nextOffset(page, limit int) int {
	if page <= 0 || limit <= 0 {
		return 0
	}
	return page * limit
}
