package commands

import (
	"context"
	"fmt"
	"image"
	_ "image/gif" // decoders for the pictures articles carry
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/graphics"
	_ "golang.org/x/image/webp"
)

const (
	// maxImageBytes is the most of a picture nom downloads.
	maxImageBytes = 10 << 20
	// maxImageWidth is the widest a picture is kept: wider ones are scaled
	// down once on arrival, rather than by the terminal on every frame.
	maxImageWidth = 1200
	// imageFetches is how many pictures download at once.
	imageFetches = 4
)

// imageLoaded carries a downloaded picture, or why there is none.
type imageLoaded struct {
	src string
	img image.Image
	err error
}

// images holds the article view's pictures by address. Update writes it and
// View reads it, both under the program's lock.
type images struct {
	client  *http.Client
	slots   chan struct{}
	loaded  map[string]image.Image
	pending map[string]bool
	failed  map[string]bool
}

func newImages(client *http.Client) *images {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &images{
		client:  client,
		slots:   make(chan struct{}, imageFetches),
		loaded:  map[string]image.Image{},
		pending: map[string]bool{},
		failed:  map[string]bool{},
	}
}

// get is Markdown.Images: the picture at src, or nil.
func (im *images) get(src string) image.Image { return im.loaded[src] }

// fetch returns commands downloading the pictures in sources that are not
// here, on their way, or known to fail.
func (im *images) fetch(sources []string, userAgent string) []limoni.Cmd {
	var cmds []limoni.Cmd
	for _, src := range sources {
		if im.loaded[src] != nil || im.pending[src] || im.failed[src] {
			continue
		}
		im.pending[src] = true
		cmds = append(cmds, im.download(src, userAgent))
	}
	return cmds
}

func (im *images) download(src, userAgent string) limoni.Cmd {
	return func(ctx context.Context) limoni.Msg {
		select {
		case im.slots <- struct{}{}:
			defer func() { <-im.slots }()
		case <-ctx.Done():
			return imageLoaded{src: src, err: ctx.Err()}
		}
		img, err := im.decode(ctx, src, userAgent)
		return imageLoaded{src: src, img: img, err: err}
	}
}

func (im *images) decode(ctx context.Context, src, userAgent string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return nil, err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	res, err := im.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", src, res.Status)
	}
	img, _, err := image.Decode(io.LimitReader(res.Body, maxImageBytes))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", src, err)
	}
	if b := img.Bounds(); b.Dx() > maxImageWidth {
		h := b.Dy() * maxImageWidth / b.Dx()
		img = graphics.ResizeImage(img, maxImageWidth, max(h, 1))
	}
	return img, nil
}

// arrived records a download.
func (im *images) arrived(msg imageLoaded) {
	delete(im.pending, msg.src)
	if msg.err != nil || msg.img == nil {
		im.failed[msg.src] = true
		return
	}
	im.loaded[msg.src] = msg.img
}

// forget drops the pictures, as the article they belong to closes: a reader
// going through a feed would otherwise keep every picture it opened.
func (im *images) forget() {
	clear(im.loaded)
	clear(im.failed)
}
