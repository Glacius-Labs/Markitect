package organization

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var renderZeroContentID snapshot.ContentID

type contextWriter struct {
	limit    Setting[int64]
	count    int64
	bytes    []byte
	sections []RenderSection
	sizes    []SectionSize
	tooLarge bool
	err      error
}

func (w *contextWriter) add(name, text string, sources ...RenderSource) {
	if w.err != nil {
		return
	}
	if err := renderText(text); err != nil {
		w.err = err
		return
	}
	if err := renderStrings(reflect.ValueOf(sources)); err != nil {
		w.err = err
		return
	}
	header := fmt.Sprintf(renderFrameOpen, name, len(text))
	size := int64(len(header)) + int64(len(text)) + int64(len(renderFrameClose))
	if w.count > math.MaxInt64-size {
		w.err = renderInvalid("rendered size overflow")
		return
	}
	section := RenderSection{Name: name, Start: w.count, End: w.count + size, TextStart: w.count + int64(len(header)), TextEnd: w.count + int64(len(header)) + int64(len(text)), Sources: sources}
	w.count += size
	w.sizes = append(w.sizes, SectionSize{Name: name, Bytes: size})
	if w.count > w.limit.Value {
		w.tooLarge = true
	}
	if !w.tooLarge {
		w.bytes = append(w.bytes, header...)
		w.bytes = append(w.bytes, text...)
		w.bytes = append(w.bytes, renderFrameClose...)
		w.sections = append(w.sections, section)
	}
}

func renderJSON(value any) (string, error) {
	if err := renderStrings(reflect.ValueOf(value)); err != nil {
		return "", err
	}
	bytes, err := json.Marshal(value)
	return string(bytes), err
}

func (w *contextWriter) json(name string, value any, sources ...RenderSource) {
	if w.err != nil {
		return
	}
	text, err := renderJSON(value)
	if err != nil {
		w.err = err
		return
	}
	w.add(name, text, sources...)
}

func hostRender(source HostSource) RenderSource {
	return RenderSource{Kind: HostRenderSource, Host: source}
}
func configRender(source SettingSource) RenderSource {
	return RenderSource{Kind: ConfigRenderSource, Config: source}
}

func (w *contextWriter) host(name string, text HostText) {
	w.add(name, text.Text, hostRender(text.Source))
}

func (w *contextWriter) reference(name string, source HostSource) {
	w.json(name, source, hostRender(source))
}
