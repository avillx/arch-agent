package mcp

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
)

type mockConfigRepo struct {
	mu      sync.Mutex
	cfgs    map[MCPServerID]ServerGatewayConfig
	loadErr error

	saveCalls   map[MCPServerID]ServerGatewayConfig
	saveErr     error
	deleteCalls []MCPServerID
	deleteErr   error
}

func newMockConfigRepo(cfgs map[MCPServerID]ServerGatewayConfig) *mockConfigRepo {
	return &mockConfigRepo{
		cfgs:      cfgs,
		saveCalls: make(map[MCPServerID]ServerGatewayConfig),
	}
}

func (r *mockConfigRepo) Load() (map[MCPServerID]ServerGatewayConfig, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return r.cfgs, nil
}

func (r *mockConfigRepo) Save(id MCPServerID, cfg ServerGatewayConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.saveCalls[id] = cfg
	if r.saveErr != nil {
		return r.saveErr
	}
	r.cfgs[id] = cfg
	return nil
}

func (r *mockConfigRepo) Delete(id MCPServerID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deleteCalls = append(r.deleteCalls, id)
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.cfgs, id)
	return nil
}

func (r *mockConfigRepo) savedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.saveCalls)
}

func (r *mockConfigRepo) deletedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.deleteCalls)
}

type mockMCPServer struct {
	id  MCPServerID
	cfg ServerGatewayConfig

	runStop      chan struct{}
	shutdownOnce sync.Once

	mu            sync.Mutex
	shutdownCalls int
	runErr        error
	err           error
}

func newMockMCPServer(id MCPServerID, cfg ServerGatewayConfig) *mockMCPServer {
	return &mockMCPServer{
		id:      id,
		cfg:     cfg,
		runStop: make(chan struct{}),
	}
}

func (m *mockMCPServer) ID() MCPServerID             { return m.id }
func (m *mockMCPServer) Config() ServerGatewayConfig { return m.cfg }
func (m *mockMCPServer) Gateway() gateway            { return nil }
func (m *mockMCPServer) Tools() []agent.Tool         { return nil }

func (m *mockMCPServer) Run(ctx context.Context) error {
	<-m.runStop

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runErr
}

func (m *mockMCPServer) Shutdown() {
	m.shutdownOnce.Do(func() {
		m.mu.Lock()
		m.shutdownCalls++
		m.mu.Unlock()
		close(m.runStop)
	})
}

func (m *mockMCPServer) Err() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

func (m *mockMCPServer) setErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *mockMCPServer) shutdownCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.shutdownCalls
}

type mockServerFactory struct {
	mu    sync.Mutex
	calls int
	byID  map[MCPServerID]MCPServer
	err   error
}

func newMockServerFactory() *mockServerFactory {
	return &mockServerFactory{byID: make(map[MCPServerID]MCPServer)}
}

func (f *mockServerFactory) newServer(ctx context.Context, id MCPServerID, cfg ServerGatewayConfig) (MCPServer, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()

	if f.err != nil {
		return nil, f.err
	}

	srv, ok := f.byID[id]
	if !ok {
		return nil, errors.New("mock factory has no server")
	}
	return srv, nil
}

func (f *mockServerFactory) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func newTestService(repo ConfigRepo, factory *mockServerFactory) *Service {
	return &Service{
		toolSvc:       tools.NewService(),
		configRepo:    repo,
		logger:        slog.New(slog.DiscardHandler),
		serverFactory: factory.newServer,
		servers:       make(map[MCPServerID]MCPServer),
	}
}

func httpConfig(url string) ServerGatewayConfig {
	return ServerGatewayConfig{HTTPGateway: &HTTPGatewayConfig{URL: url}}
}

func containsServer(servers []MCPServer, want MCPServer) bool {
	return slices.Contains(servers, want)
}

func serverConnected(t *testing.T, toolSvc *tools.Service, id MCPServerID) bool {
	t.Helper()

	_, err := toolSvc.ToolServers(string(id))
	return err == nil
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func TestNewServiceEmptyRepo(t *testing.T) {
	svc, err := NewService(
		context.Background(),
		tools.NewService(),
		newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{}),
		slog.New(slog.DiscardHandler),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if got := len(svc.List()); got != 0 {
		t.Fatalf("expected no servers, got %d", got)
	}
}

func TestNewServiceLoadError(t *testing.T) {
	wantErr := errors.New("load failed")
	repo := newMockConfigRepo(nil)
	repo.loadErr = wantErr

	_, err := NewService(
		context.Background(),
		tools.NewService(),
		repo,
		slog.New(slog.DiscardHandler),
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("NewService() error = %v, want %v", err, wantErr)
	}
}

func TestConnectServerRegistersAndDisconnects(t *testing.T) {
	svc := newTestService(newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{}), newMockServerFactory())
	srv := newMockMCPServer("srv", httpConfig("http://example.com"))
	svc.serverFactory = func(context.Context, MCPServerID, ServerGatewayConfig) (MCPServer, error) {
		return srv, nil
	}

	if err := svc.connectServer(context.Background(), "srv", httpConfig("http://example.com")); err != nil {
		t.Fatalf("connectServer() error = %v", err)
	}
	if srv.shutdownCount() != 0 {
		t.Fatalf("expected no shutdown on connect, got %d", srv.shutdownCount())
	}
	if !serverConnected(t, svc.toolSvc, "srv") {
		t.Fatal("expected server to be connected to tool service")
	}

	srv.Shutdown()
	waitFor(t, func() bool {
		return !serverConnected(t, svc.toolSvc, "srv")
	})
}

func TestConnectServerInitError(t *testing.T) {
	wantErr := errors.New("init failed")
	factory := newMockServerFactory()
	factory.err = wantErr

	svc := newTestService(newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{}), factory)

	err := svc.connectServer(context.Background(), "srv", httpConfig("http://example.com"))
	if !errors.Is(err, wantErr) {
		t.Fatalf("connectServer() error = %v, want %v", err, wantErr)
	}
	if got := len(svc.List()); got != 0 {
		t.Fatalf("expected no servers, got %d", got)
	}
}

func TestConnectServerToolServiceConflict(t *testing.T) {
	svc := newTestService(newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{}), newMockServerFactory())
	existing := newMockMCPServer("srv", httpConfig("http://example.com"))
	if err := svc.toolSvc.Connect("srv", existing); err != nil {
		t.Fatalf("prepare tool service: %v", err)
	}

	factory := newMockServerFactory()
	factory.byID["srv"] = newMockMCPServer("srv", httpConfig("http://example.com"))
	svc.serverFactory = factory.newServer

	err := svc.connectServer(context.Background(), "srv", httpConfig("http://example.com"))
	if err == nil {
		t.Fatal("expected tool service conflict error")
	}
	if got := len(svc.List()); got != 0 {
		t.Fatalf("expected no servers, got %d", got)
	}
}

func TestReloadAddsNewServers(t *testing.T) {
	factory := newMockServerFactory()
	srv := newMockMCPServer("srv", httpConfig("http://example.com"))
	factory.byID["srv"] = srv

	svc := newTestService(newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{
		"srv": httpConfig("http://example.com"),
	}), factory)

	if err := svc.Reload(context.Background()); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	servers := svc.List()
	if !containsServer(servers, srv) {
		t.Fatalf("expected server in list, got %v", servers)
	}
	if srv.shutdownCount() != 0 {
		t.Fatalf("expected no shutdown, got %d", srv.shutdownCount())
	}
	if !serverConnected(t, svc.toolSvc, "srv") {
		t.Fatal("expected server to be connected to tool service")
	}
}

func TestReloadRemovesDeletedServers(t *testing.T) {
	factory := newMockServerFactory()
	oldSrv := newMockMCPServer("srv", httpConfig("http://example.com"))

	svc := newTestService(newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{}), factory)
	svc.servers["srv"] = oldSrv

	if err := svc.Reload(context.Background()); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	if oldSrv.shutdownCount() != 1 {
		t.Fatalf("expected one shutdown, got %d", oldSrv.shutdownCount())
	}
	if got := len(svc.List()); got != 0 {
		t.Fatalf("expected no servers, got %d", got)
	}
	if factory.callCount() != 0 {
		t.Fatalf("expected no new server creation, got %d", factory.callCount())
	}
}

func TestReloadUpdatesChangedConfig(t *testing.T) {
	oldCfg := httpConfig("http://old")
	newCfg := httpConfig("http://new")

	oldSrv := newMockMCPServer("srv", oldCfg)
	newSrv := newMockMCPServer("srv", newCfg)

	factory := newMockServerFactory()
	factory.byID["srv"] = newSrv

	svc := newTestService(newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{
		"srv": newCfg,
	}), factory)
	svc.servers["srv"] = oldSrv

	if err := svc.Reload(context.Background()); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	if oldSrv.shutdownCount() != 1 {
		t.Fatalf("expected old server shutdown once, got %d", oldSrv.shutdownCount())
	}
	servers := svc.List()
	if !containsServer(servers, newSrv) {
		t.Fatalf("expected new server in list, got %v", servers)
	}
	if containsServer(servers, oldSrv) {
		t.Fatal("expected old server removed from list")
	}
	if factory.callCount() != 1 {
		t.Fatalf("expected one server creation, got %d", factory.callCount())
	}
}

func TestReloadKeepsUnchangedConfig(t *testing.T) {
	cfg := httpConfig("http://example.com")
	srv := newMockMCPServer("srv", cfg)

	factory := newMockServerFactory()

	svc := newTestService(newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{
		"srv": cfg,
	}), factory)
	svc.servers["srv"] = srv

	if err := svc.Reload(context.Background()); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	if srv.shutdownCount() != 0 {
		t.Fatalf("expected no shutdown, got %d", srv.shutdownCount())
	}
	servers := svc.List()
	if !containsServer(servers, srv) {
		t.Fatalf("expected server to stay in list, got %v", servers)
	}
	if factory.callCount() != 0 {
		t.Fatalf("expected no server creation, got %d", factory.callCount())
	}
}

func TestReloadLoadError(t *testing.T) {
	wantErr := errors.New("load failed")
	repo := newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{
		"srv": httpConfig("http://example.com"),
	})
	repo.loadErr = wantErr

	svc := newTestService(repo, newMockServerFactory())

	err := svc.Reload(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Reload() error = %v, want %v", err, wantErr)
	}
	if got := len(svc.List()); got != 0 {
		t.Fatalf("expected no servers, got %d", got)
	}
}

func TestLoadConnectsAllServers(t *testing.T) {
	factory := newMockServerFactory()
	srvA := newMockMCPServer("a", httpConfig("http://a"))
	srvB := newMockMCPServer("b", httpConfig("http://b"))
	factory.byID["a"] = srvA
	factory.byID["b"] = srvB

	svc := newTestService(newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{
		"a": httpConfig("http://a"),
		"b": httpConfig("http://b"),
	}), factory)

	svc.load(context.Background())

	servers := svc.List()
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	if !containsServer(servers, srvA) || !containsServer(servers, srvB) {
		t.Fatalf("expected both servers in list, got %v", servers)
	}
}

func TestSetServerReplacesExisting(t *testing.T) {
	oldCfg := httpConfig("http://old")
	newCfg := httpConfig("http://new")

	oldSrv := newMockMCPServer("srv", oldCfg)
	newSrv := newMockMCPServer("srv", newCfg)

	factory := newMockServerFactory()
	factory.byID["srv"] = newSrv

	repo := newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{})
	svc := newTestService(repo, factory)
	svc.servers["srv"] = oldSrv

	if err := svc.SetServer(context.Background(), "srv", newCfg); err != nil {
		t.Fatalf("SetServer() error = %v", err)
	}

	if oldSrv.shutdownCount() != 1 {
		t.Fatalf("expected old server shutdown once, got %d", oldSrv.shutdownCount())
	}
	servers := svc.List()
	if !containsServer(servers, newSrv) {
		t.Fatalf("expected new server in list, got %v", servers)
	}
	if repo.savedCount() != 1 {
		t.Fatalf("expected one save, got %d", repo.savedCount())
	}
	if !repo.cfgs["srv"].Equals(newCfg) {
		t.Fatalf("expected saved config %v, got %v", newCfg, repo.cfgs["srv"])
	}
}

func TestSetServerAddsNew(t *testing.T) {
	cfg := httpConfig("http://example.com")
	srv := newMockMCPServer("srv", cfg)

	factory := newMockServerFactory()
	factory.byID["srv"] = srv

	repo := newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{})
	svc := newTestService(repo, factory)

	if err := svc.SetServer(context.Background(), "srv", cfg); err != nil {
		t.Fatalf("SetServer() error = %v", err)
	}

	servers := svc.List()
	if !containsServer(servers, srv) {
		t.Fatalf("expected server in list, got %v", servers)
	}
	if repo.savedCount() != 1 {
		t.Fatalf("expected one save, got %d", repo.savedCount())
	}
}

func TestSetServerInitError(t *testing.T) {
	wantErr := errors.New("init failed")
	factory := newMockServerFactory()
	factory.err = wantErr

	oldSrv := newMockMCPServer("srv", httpConfig("http://old"))

	repo := newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{})
	svc := newTestService(repo, factory)
	svc.servers["srv"] = oldSrv

	err := svc.SetServer(context.Background(), "srv", httpConfig("http://new"))
	if !errors.Is(err, wantErr) {
		t.Fatalf("SetServer() error = %v, want %v", err, wantErr)
	}

	if oldSrv.shutdownCount() != 0 {
		t.Fatalf("expected no shutdown on failed set, got %d", oldSrv.shutdownCount())
	}
	if repo.savedCount() != 0 {
		t.Fatalf("expected no save on failed set, got %d", repo.savedCount())
	}
	servers := svc.List()
	if !containsServer(servers, oldSrv) {
		t.Fatalf("expected old server to remain in list, got %v", servers)
	}
}

func TestDeleteServer(t *testing.T) {
	repo := newMockConfigRepo(map[MCPServerID]ServerGatewayConfig{
		"srv": httpConfig("http://example.com"),
	})
	svc := newTestService(repo, newMockServerFactory())

	if err := svc.DeleteServer("srv"); err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}
	if repo.deletedCount() != 1 {
		t.Fatalf("expected one delete, got %d", repo.deletedCount())
	}
}
