package service

const pageSize = 10

// effectiveStatus is the word's status with the pending draft applied.
func effectiveStatus(u *User, base string) string {
	if u.State.List != nil {
		if t, ok := u.State.List.Draft[base]; ok {
			return t
		}
	}
	if w, ok := u.Words[base]; ok {
		return w.Status
	}
	return StatusNew
}

// pageBounds returns the [start,end) slice bounds for the given page, the total
// page count (min 1), and the clamped page index.
func pageBounds(n, page int) (start, end, pages, clamped int) {
	pages = (n + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start = page * pageSize
	end = start + pageSize
	if end > n {
		end = n
	}
	return start, end, pages, page
}
