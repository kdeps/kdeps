package session

// Close is a helper method available only in test builds to simplify resource cleanup.
// It closes the underlying *sql.DB if it is non-nil.
func (r *PklResourceReader) Close() error {
	if r == nil || r.DB == nil {
		return nil
	}
	return r.DB.Close()
}
