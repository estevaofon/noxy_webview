# noxy_webview

Uma janela nativa para uma página local, como extensão por processo do
[Noxy](https://github.com/noxylang/noxy): WebKitGTK no Linux, WebView2 no
Windows, WKWebView no macOS, via [webview_go](https://github.com/webview/webview_go).
Feita para o [Noxy Editor](https://github.com/noxylang/Noxy-Editor), serve
para qualquer programa Noxy que sirva uma página em `127.0.0.1` e queira uma
janela sem barra de endereço.

## Instalação

    noxy --get github.com/noxylang/noxy_webview

`noxy --get` baixa o binário da sua plataforma para `bin/` e grava os hashes
em `noxy.sum`. Requer Noxy 0.25.0 ou mais novo. Em runtime, o Linux precisa
de `libwebkit2gtk-4.1` (presente em desktops GNOME); o Windows, do runtime
WebView2 (incluído no Windows 11).

## API

```noxy
use github_com.noxylang.noxy_webview.noxy_webview as webview

webview.open("Minha página", 1200, 800, "http://127.0.0.1:8080/")
webview.set_title("Minha página — carregada")
webview.wait()          // bloqueia até a janela fechar
```

| Função | Efeito |
|---|---|
| `open(title, width, height, url)` | Abre a única janela do processo. Erro se já aberta, já fechada ou tamanho não positivo |
| `wait()` | Bloqueia até a janela fechar, pelo X ou por `close` |
| `close()` | Pede o fechamento; no-op depois de fechada |
| `set_title(title)` | Muda o título; erro depois de fechada |

Toda falha é um erro de runtime `extension 'webview' failed: <motivo>`,
capturável com `call_result`. `NOXY_WEBVIEW_DEBUG=1` no ambiente do `noxy`
abre a janela com o inspetor do WebKit.

## Ciclo de vida

O SDK (`noxyplugin`) serve stdin/stdout numa goroutine; a thread principal
espera `open` e roda o loop da janela, que exige a main. Quando o loop volta,
a janela é destruída e todo `wait` pendente retorna. O processo vive até o
host fechar o stdin. A máquina de estados fica no pacote `window/`, sem
dependência da biblioteca webview, para `go test ./window/` rodar em qualquer
máquina.

## Compilar localmente

Linux:

    sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev
    sh release/build.sh webview
    mkdir -p bin && cp dist/noxy-plugin-webview-linux-amd64 bin/

O `webview_go` publicado ainda pede `webkit2gtk-4.0` no `pkg-config`, que as
distros atuais não têm (PR webview_go#62, aberto). `release/build.sh` contorna
com um diretório temporário onde `webkit2gtk-4.0.pc` e
`javascriptcoregtk-4.0.pc` apontam para os arquivos 4.1, posto no
`PKG_CONFIG_PATH` só durante o build.

Para usar um checkout num projeto sem release, linke o diretório em
`<projeto>/noxy_libs/github_com/noxylang/noxy_webview`; sem entrada em
`noxy.sum` a VM avisa uma vez e roda. Então `noxy examples/smoke.nx` (a partir
do projeto) abre uma janela por meio segundo e imprime `ok`.

## Release

Push de uma tag `vX.Y.Z`. O workflow compila em um runner por plataforma
(cgo em todas), junta os checksums em `checksums.txt` e publica os assets com
os nomes de `[binaries]` em `noxy_ext.toml`.
