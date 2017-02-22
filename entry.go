package microblob

// Entry associates a key with a section in a file specified by offset and length.
type Entry struct {
	Key    string `json:"k"`
	Offset int64  `json:"o"`
	Length int64  `json:"l"`
}
