// noxy_webview — uma janela nativa para uma pagina local, empacotada como
// extensao por processo do Noxy (kind = "process" em noxy_ext.toml).
//
// O SDK (noxyplugin) serve stdin/stdout numa goroutine; a thread principal
// espera webview_open e entao roda o loop da janela, que exige a main
// (webview_go trava a thread principal no init). Uma janela por processo.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/noxylang/noxy/sdk/noxyplugin"
	webview "github.com/webview/webview_go"

	"github.com/noxylang/noxy_webview/window"
)

// native adapta webview.WebView a window.Native: SetSize sem hint.
type native struct{ webview.WebView }

func (n native) SetSize(w, h int) { n.WebView.SetSize(w, h, webview.HintNone) }

func main() {
	st := window.New()
	p := noxyplugin.New()
	p.Handle("webview_open", noxyplugin.Func4(func(ctx context.Context, title string, width, height int64, url string) (any, error) {
		if width <= 0 || height <= 0 {
			return nil, fmt.Errorf("width and height must be positive, got %dx%d", width, height)
		}
		if url == "" {
			return nil, fmt.Errorf("url must not be empty")
		}
		return nil, st.Open(ctx, window.OpenRequest{Title: title, Width: int(width), Height: int(height), URL: url})
	}))
	p.Handle("webview_wait", noxyplugin.Func0(func(ctx context.Context) (any, error) { return nil, st.Wait(ctx) }))
	p.Handle("webview_close", noxyplugin.Func0(func(ctx context.Context) (any, error) { return nil, st.Close() }))
	p.Handle("webview_set_title", noxyplugin.Func1(func(ctx context.Context, title string) (any, error) { return nil, st.SetTitle(title) }))
	go p.Main() // sai do processo (os.Exit) quando o host fecha o stdin

	req := <-st.OpenRequests()
	debug := os.Getenv("NOXY_WEBVIEW_DEBUG") == "1" // inspetor do WebKit
	st.Serve(req, func() window.Native { return native{webview.New(debug)} })
	select {} // a janela fechou; o processo vive ate o host fechar o stdin
}
