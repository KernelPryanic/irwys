package irwys

type Options struct {
	minLength uint16
	maxLength uint16
	timeout   int16
	capacity  uint16
	dbPath    string
	replyPath string
	verbose   bool
}

func NewOptions(
	minLength uint16,
	maxLength uint16,
	timeout int16,
	capacity uint16,
	dbPath string,
	replyPath string,
	verbose bool,
) Options {
	o := Options{
		minLength, maxLength, timeout,
		capacity, dbPath, replyPath, verbose,
	}
	return o
}
