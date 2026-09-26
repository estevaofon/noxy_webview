#!/usr/bin/env sh
# Builds this platform's plugin binary into dist/ and its sha256 line. A
# biblioteca webview e cgo em todo OS, entao cada OS compila o seu: o
# workflow roda isto num runner por plataforma e junta os checksums.
#
# No Linux, webview_go ainda pede webkit2gtk-4.0 no pkg-config (PR
# webview_go#62 troca para 4.1, ainda aberto) e as distros atuais so tem a
# 4.1. O shim abaixo cria um diretorio temporario onde os nomes 4.0 apontam
# para os arquivos .pc da 4.1, e o poe no PKG_CONFIG_PATH so para este build.
set -eu
NAME="${1:?usage: build.sh <extension-name> [GOARCH]}"
ARCH="${2:-$(go env GOARCH)}"
OS="$(go env GOOS)"
ext=""; [ "$OS" = windows ] && ext=".exe"
if [ "$OS" = linux ] && ! pkg-config --exists webkit2gtk-4.0 && pkg-config --exists webkit2gtk-4.1; then
  shim="$(mktemp -d)"
  for lib in webkit2gtk javascriptcoregtk; do
    ln -s "$(pkg-config --variable=pcfiledir "$lib-4.1")/$lib-4.1.pc" "$shim/$lib-4.0.pc"
  done
  export PKG_CONFIG_PATH="$shim${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
fi
mkdir -p dist
CGO_ENABLED=1 GOARCH="$ARCH" go build -trimpath -ldflags=-s -o "dist/noxy-plugin-$NAME-$OS-$ARCH$ext" .
(cd dist && sha256sum -- "noxy-plugin-$NAME-$OS-$ARCH$ext" > "checksums-$OS-$ARCH.txt")
echo "dist/noxy-plugin-$NAME-$OS-$ARCH$ext"
