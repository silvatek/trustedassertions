package web

import (
	"net/http"
	"strings"

	"github.com/skip2/go-qrcode"
)

func SharePageWebHandler(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	kind := r.URL.Query().Get("type")

	var prefix string
	host := r.Host
	if strings.Contains(host, "localhost") {
		prefix = "http://" + host
	} else {
		prefix = "https://" + host
	}

	data := struct {
		Url     string
		QrCode  string
		HashUri string
	}{
		Url:     prefix + "/web/" + kind + "s/" + hash,
		QrCode:  prefix + "/web/qrcode?hash=" + hash + "&type=" + kind,
		HashUri: "hash://sha256/" + hash + "?type=" + kind,
	}

	RenderWebPage("sharepage", data, nil, w, r)
}

func qrCodeGenerator(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	kind := r.URL.Query().Get("type")

	var prefix string
	host := r.Host
	if strings.Contains(host, "localhost") {
		prefix = "http://" + host
	} else {
		prefix = "https://" + host
	}

	uri := prefix + "/web/" + kind + "s/" + hash

	headers := w.Header()
	headers.Add("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)

	q, _ := qrcode.New(uri, qrcode.High)
	q.Write(320, w)
}
