// Package window guarda o estado da janela unica do processo sem tocar na
// biblioteca webview: main.go injeta a implementacao nativa e os testes
// injetam uma falsa. Regras: uma janela por processo; open repetido ou
// depois de fechada e erro; wait, close e set_title antes de open sao erro;
// close depois de fechada e no-op.
package window

import (
	"context"
	"errors"
	"sync"
)

// Native e o que a janela nativa precisa oferecer; main.go adapta
// webview.WebView a esta interface.
type Native interface {
	SetTitle(title string)
	SetSize(w, h int)
	Navigate(url string)
	Run()
	Terminate()
	Dispatch(f func())
	Destroy()
}

// OpenRequest e o pedido que a thread principal espera para criar a janela.
type OpenRequest struct {
	Title         string
	Width, Height int
	URL           string
}

var (
	ErrAlreadyOpen = errors.New("window already open")
	ErrClosed      = errors.New("window was closed")
	ErrNotOpen     = errors.New("window is not open: call webview.open first")
)

// State e a maquina de estados: aberta, pronta (native criado) e fechada.
type State struct {
	mu      sync.Mutex
	opened  bool
	closed  bool
	native  Native
	openReq chan OpenRequest // consumido pela thread principal
	ready   chan struct{}    // fechado quando native existe, antes de Run
	done    chan struct{}    // fechado quando Run volta
}

func New() *State {
	return &State{openReq: make(chan OpenRequest, 1), ready: make(chan struct{}), done: make(chan struct{})}
}

// Open e o handler de webview_open: entrega o pedido a thread principal e
// espera a janela existir, ou o contexto acabar.
func (s *State) Open(ctx context.Context, req OpenRequest) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}
	if s.opened {
		s.mu.Unlock()
		return ErrAlreadyOpen
	}
	s.opened = true
	s.mu.Unlock()
	s.openReq <- req
	select {
	case <-s.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// OpenRequests e lido pela thread principal: o unico pedido de abertura.
func (s *State) OpenRequests() <-chan OpenRequest { return s.openReq }

// Serve roda na thread principal: cria a janela com create, marca pronta,
// roda o loop ate a janela fechar, destroi a janela e so entao libera Wait.
// Bloqueia ate o fim.
func (s *State) Serve(req OpenRequest, create func() Native) {
	n := create()
	n.SetTitle(req.Title)
	n.SetSize(req.Width, req.Height)
	n.Navigate(req.URL)
	s.mu.Lock()
	s.native = n
	s.mu.Unlock()
	close(s.ready)
	n.Run()
	s.mu.Lock()
	s.closed = true // a partir daqui Close e no-op e SetTitle e erro
	s.mu.Unlock()
	n.Destroy()
	close(s.done) // quem acorda em Wait ja ve a janela destruida
}

// Wait e o handler de webview_wait: bloqueia ate a janela fechar.
func (s *State) Wait(ctx context.Context) error {
	s.mu.Lock()
	opened := s.opened
	s.mu.Unlock()
	if !opened {
		return ErrNotOpen
	}
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// live espera a janela existir e devolve o native, ou nil se ja fechou.
func (s *State) live() (Native, error) {
	s.mu.Lock()
	opened := s.opened
	s.mu.Unlock()
	if !opened {
		return nil, ErrNotOpen
	}
	<-s.ready
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, nil
	}
	return s.native, nil
}

// Close e o handler de webview_close: pede o fim do loop; no-op se ja fechou.
// Terminate e seguro de qualquer thread, mas Dispatch mantem tudo na UI.
func (s *State) Close() error {
	n, err := s.live()
	if err != nil || n == nil {
		return err
	}
	n.Dispatch(n.Terminate)
	return nil
}

// SetTitle e o handler de webview_set_title; erro depois de fechada.
func (s *State) SetTitle(title string) error {
	n, err := s.live()
	if err != nil {
		return err
	}
	if n == nil {
		return ErrClosed
	}
	n.Dispatch(func() { n.SetTitle(title) })
	return nil
}
