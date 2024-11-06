package web

import (
	"io"
	"net/http"
	"strings"

	"github.com/skip2/go-qrcode"
	"silvatek.uk/trustedassertions/internal/appcontext"
)

func SharePageWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	hash := r.URL.Query().Get("hash")
	kind := r.URL.Query().Get("type")

	data := struct {
		Url     string
		QrCode  string
		HashUri string
	}{
		Url:     server(r.Host) + "/web/" + kind + "s/" + hash,
		QrCode:  server(r.Host) + "/web/qrcode?hash=" + hash + "&type=" + kind,
		HashUri: "hash://sha256/" + hash + "?type=" + kind,
	}

	RenderWebPage(ctx, "sharepage", data, nil, w, r)
}

func qrCodeGenerator(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	kind := r.URL.Query().Get("type")
	host := r.Host

	headers := w.Header()
	headers.Add("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)

	writeQrCode(host, hash, kind, w)
}

func writeQrCode(host string, hash string, kind string, w io.Writer) {
	uri := server(host) + "/web/" + kind + "s/" + hash

	q, _ := qrcode.New(uri, qrcode.High)
	q.Write(320, w)
}

func server(host string) string {
	if strings.Contains(host, "localhost") {
		return "http://" + host
	} else {
		return "https://" + host
	}
}
