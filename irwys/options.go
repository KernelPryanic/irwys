package irwys

// Options structure.
type Options struct {
	minWords  uint16
	maxWords  uint16
	timeout   int16
	timeStart uint8
	timeEnd   uint8
	capacity  uint16
	dbPath    string
	replyPath string
	verbose   bool
}

// NewOptions creates an object of NewOptions structure.
func NewOptions(
	minWords uint16,
	maxWords uint16,
	timeout int16,
	timeStart uint8,
	timeEnd uint8,
	capacity uint16,
	dbPath string,
	replyPath string,
	verbose bool,
) Options {
	o := Options{
		minWords, maxWords, timeout, timeStart, timeEnd,
		capacity, dbPath, replyPath, verbose,
	}
	return o
}
