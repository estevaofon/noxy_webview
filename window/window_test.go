package window

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fake e uma janela sem tela: Run bloqueia ate Terminate; Dispatch executa
// na hora, como se ja estivesse na thread principal.
type fake struct {
	mu     sync.Mutex
	titles []string
	size   [2]int
	url    string
	quit   chan struct{}
	once   sync.Once
	killed bool
}

func newFake() *fake { return &fake{quit: make(chan struct{})} }

func (f *fake) SetTitle(t string)  { f.mu.Lock(); f.titles = append(f.titles, t); f.mu.Unlock() }
func (f *fake) SetSize(w, h int)   { f.size = [2]int{w, h} }
func (f *fake) Navigate(u string)  { f.url = u }
func (f *fake) Run()               { <-f.quit }
func (f *fake) Terminate()         { f.once.Do(func() { close(f.quit) }) }
func (f *fake) Dispatch(fn func()) { fn() }
func (f *fake) Destroy()           { f.killed = true }

// start sobe a thread principal falsa: espera o pedido e serve.
func start(t *testing.T, s *State, f *fake) {
	t.Helper()
	go func() {
		req := <-s.OpenRequests()
		s.Serve(req, func() Native { return f })
	}()
}

func TestBeforeOpenEverythingFails(t *testing.T) {
	s := New()
	if err := s.Wait(context.Background()); !errors.Is(err, ErrNotOpen) {
		t.Fatalf("Wait antes de Open: %v", err)
	}
	if err := s.Close(); !errors.Is(err, ErrNotOpen) {
		t.Fatalf("Close antes de Open: %v", err)
	}
	if err := s.SetTitle("x"); !errors.Is(err, ErrNotOpen) {
		t.Fatalf("SetTitle antes de Open: %v", err)
	}
}

func TestOpenConfiguresAndWaitReturnsOnClose(t *testing.T) {
	s, f := New(), newFake()
	start(t, s, f)
	req := OpenRequest{Title: "Noxy Editor", Width: 1200, Height: 800, URL: "http://127.0.0.1:1/?t=x"}
	if err := s.Open(context.Background(), req); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if f.titles[0] != "Noxy Editor" || f.size != [2]int{1200, 800} || f.url != req.URL {
		t.Fatalf("janela configurada errado: %+v %v %q", f.titles, f.size, f.url)
	}
	if err := s.Open(context.Background(), req); !errors.Is(err, ErrAlreadyOpen) {
		t.Fatalf("segundo Open: %v", err)
	}
	if err := s.SetTitle("main.nx — Noxy Editor"); err != nil || f.titles[len(f.titles)-1] != "main.nx — Noxy Editor" {
		t.Fatalf("SetTitle: %v %v", err, f.titles)
	}
	waited := make(chan error, 1)
	go func() { waited <- s.Wait(context.Background()) }()
	select {
	case err := <-waited:
		t.Fatalf("Wait voltou antes de fechar: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	select {
	case err := <-waited:
		if err != nil {
			t.Fatalf("Wait depois de Close: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Wait nao destravou com Close")
	}
	f.mu.Lock()
	killed := f.killed
	f.mu.Unlock()
	if !killed {
		t.Fatal("Destroy nao foi chamado antes de Wait voltar")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close depois de fechada deve ser no-op: %v", err)
	}
	if err := s.SetTitle("x"); !errors.Is(err, ErrClosed) {
		t.Fatalf("SetTitle depois de fechada: %v", err)
	}
	if err := s.Open(context.Background(), req); !errors.Is(err, ErrClosed) {
		t.Fatalf("Open depois de fechada: %v", err)
	}
	if err := s.Wait(context.Background()); err != nil {
		t.Fatalf("Wait depois de fechada volta na hora: %v", err)
	}
}

func TestWaitHonoursContext(t *testing.T) {
	s, f := New(), newFake()
	start(t, s, f)
	if err := s.Open(context.Background(), OpenRequest{Title: "t", Width: 1, Height: 1, URL: "u"}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := s.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait com contexto expirado: %v", err)
	}
	f.Terminate()
}

func TestOpenHonoursContextWhenMainNeverServes(t *testing.T) {
	s := New()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := s.Open(ctx, OpenRequest{Title: "t", Width: 1, Height: 1, URL: "u"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Open sem thread principal: %v", err)
	}
}
