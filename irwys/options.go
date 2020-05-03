package irwys

// Options structure.
type Options struct {
	minLength uint16
	maxLength uint16
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
	minLength uint16,
	maxLength uint16,
	timeout int16,
	timeStart uint8,
	timeEnd uint8,
	capacity uint16,
	dbPath string,
	replyPath string,
	verbose bool,
) Options {
	o := Options{
		minLength, maxLength, timeout, timeStart, timeEnd,
		capacity, dbPath, replyPath, verbose,
	}
	return o
}
