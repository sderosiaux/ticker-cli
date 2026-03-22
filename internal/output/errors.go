package output

import "errors"

// Sentinel errors for output formatting.
var (
	ErrUnsupportedFormat   = errors.New("unsupported format")
	ErrUnsupportedDataType = errors.New("unsupported data type")
)
