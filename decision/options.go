package decision

import (
	"crypto/rand"
	"encoding/hex"
)

// Image is a picture sent alongside the state. Write its Ref wherever the
// image belongs in the state, a question's instructions or its criteria, and
// the model sees the image in that place. Every image passed to
// [WithImages] must be referenced at least once.
type Image struct {
	// Ref is the placeholder string standing in for the image in text.
	Ref string
	// Format is the image format: "png", "jpg" or "jpeg".
	Format string
	// Data is the raw image file contents.
	Data []byte
}

// NewImage wraps raw image bytes with a freshly generated unique Ref. Format
// is "png", "jpg" or "jpeg".
func NewImage(format string, data []byte) Image {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return Image{Ref: "img-" + hex.EncodeToString(b[:]), Format: format, Data: data}
}

// CallOptions holds the per-call settings collected from [CallOption]s.
// Vendor packages build it with [ApplyOptions].
type CallOptions struct {
	// Images are the images referenced from the state or questions.
	Images []Image
}

// CallOption customises a single [Decider.Decide] call.
type CallOption func(*CallOptions)

// WithImages sends images with the call. Image input is an early feature of
// the providers that support it.
func WithImages(images ...Image) CallOption {
	return func(o *CallOptions) {
		o.Images = append(o.Images, images...)
	}
}

// ApplyOptions folds opts into a [CallOptions].
func ApplyOptions(opts []CallOption) CallOptions {
	var o CallOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
