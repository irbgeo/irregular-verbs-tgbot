package bot

// dispatchArgs bundles dispatch's arguments beyond ctx.
type dispatchArgs struct {
	UserID int64
	Kind   string
	Value  string
}
