package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/repository/sqlite"
	"github.com/NexzaDev/argus-asm/internal/service"
)

var fixedNow = time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)

type fixture struct {
	t     *testing.T
	ctx   context.Context
	store *sqlite.Store
	svc   *service.Service
	now   time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "argus.db"), true)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	f := &fixture{t: t, ctx: ctx, store: store, now: fixedNow}
	f.svc = service.New(store, func() time.Time { return f.now })
	return f
}

func (f *fixture) bootstrap() domain.Actor {
	f.t.Helper()
	actor, err := f.svc.ResolveActor(f.ctx, "")
	if err != nil {
		f.t.Fatalf("bootstrap actor: %v", err)
	}
	return actor
}

func (f *fixture) admin() domain.Actor {
	f.t.Helper()
	admin := f.bootstrap()
	if _, err := f.svc.CreateUser(f.ctx, admin, "root", "admin"); err != nil {
		f.t.Fatalf("create admin: %v", err)
	}
	actor, err := f.svc.ResolveActor(f.ctx, "root")
	if err != nil {
		f.t.Fatalf("resolve admin: %v", err)
	}
	return actor
}

func (f *fixture) mustUser(admin domain.Actor, name, role string) domain.Actor {
	f.t.Helper()
	if _, err := f.svc.CreateUser(f.ctx, admin, name, role); err != nil {
		f.t.Fatalf("create %s: %v", name, err)
	}
	actor, err := f.svc.ResolveActor(f.ctx, name)
	if err != nil {
		f.t.Fatalf("resolve %s: %v", name, err)
	}
	return actor
}

func (f *fixture) seedInventory(admin domain.Actor) {
	f.t.Helper()
	if _, err := f.svc.AddScope(f.ctx, admin, "example.test"); err != nil {
		f.t.Fatalf("scope: %v", err)
	}
	if _, err := f.svc.AddScope(f.ctx, admin, "192.0.2.0/24"); err != nil {
		f.t.Fatalf("cidr scope: %v", err)
	}
	for _, value := range []string{"www.example.test", "192.0.2.10"} {
		if _, err := f.svc.AddAsset(f.ctx, admin, value); err != nil {
			f.t.Fatalf("asset %s: %v", value, err)
		}
	}
}

func isErr(err, target error) bool {
	return errors.Is(err, target)
}

func TestBootstrapActorIsAdminUntilUsersExist(t *testing.T) {
	f := newFixture(t)
	actor := f.bootstrap()
	if actor.Name != domain.BootstrapActorName || actor.Role != domain.RoleAdmin {
		t.Fatalf("bootstrap = %+v", actor)
	}
	if _, err := f.svc.ResolveActor(f.ctx, "alice"); !isErr(err, domain.ErrForbidden) {
		t.Fatalf("unknown actor before users: err = %v, want forbidden", err)
	}
}

func TestFirstUserMustBeAdministrator(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.CreateUser(f.ctx, f.bootstrap(), "vera", "viewer"); !isErr(err, domain.ErrValidation) {
		t.Fatalf("bootstrap creating a viewer: err = %v, want validation", err)
	}
	if _, err := f.svc.CreateUser(f.ctx, f.bootstrap(), "root", "admin"); err != nil {
		t.Fatalf("bootstrap creating the first admin: %v", err)
	}
}

func TestOnceUsersExistActorIsRequired(t *testing.T) {
	f := newFixture(t)
	f.admin()
	if _, err := f.svc.ResolveActor(f.ctx, ""); !isErr(err, domain.ErrForbidden) {
		t.Fatalf("missing actor: err = %v, want forbidden", err)
	}
	if _, err := f.svc.ResolveActor(f.ctx, "local"); !isErr(err, domain.ErrForbidden) {
		t.Fatalf("bootstrap after users exist: err = %v, want forbidden", err)
	}
}

func TestRolesGateMutations(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	viewer := f.mustUser(admin, "vera", "viewer")
	analyst := f.mustUser(admin, "ana", "analyst")

	if _, err := f.svc.AddScope(f.ctx, viewer, "example.test"); !isErr(err, domain.ErrForbidden) {
		t.Fatalf("viewer adding scope: err = %v, want forbidden", err)
	}
	if _, err := f.svc.AddScope(f.ctx, analyst, "example.test"); !isErr(err, domain.ErrForbidden) {
		t.Fatalf("analyst adding scope: err = %v, want forbidden", err)
	}
	f.seedInventory(admin)
	if _, err := f.svc.ListAssets(f.ctx, viewer); err != nil {
		t.Fatalf("viewer reading assets: %v", err)
	}
	if _, err := f.svc.AddAsset(f.ctx, viewer, "api.example.test"); !isErr(err, domain.ErrForbidden) {
		t.Fatalf("viewer adding asset: err = %v, want forbidden", err)
	}
	if _, err := f.svc.AddAsset(f.ctx, analyst, "api.example.test"); err != nil {
		t.Fatalf("analyst adding asset: %v", err)
	}
	if _, _, err := f.svc.AuditLog(f.ctx, analyst); !isErr(err, domain.ErrForbidden) {
		t.Fatalf("analyst reading audit: err = %v, want forbidden", err)
	}
}

func TestAssetOutsideScopeIsRejected(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	if _, err := f.svc.AddScope(f.ctx, admin, "example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AddAsset(f.ctx, admin, "203.0.113.99"); !isErr(err, domain.ErrOutOfScope) {
		t.Fatalf("out-of-scope asset: err = %v, want ErrOutOfScope", err)
	}
}

func TestFindingsImportIsAllOrNothingAndNotifies(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)

	bad := "asset,title,severity,source\nwww.example.test,TLS expired,high,scanner-a\n8.8.4.4,Open port,low,scanner-a\n"
	report, err := f.svc.ImportFindings(f.ctx, admin, "csv", "scanner-a", strings.NewReader(bad), false)
	if !isErr(err, domain.ErrValidation) {
		t.Fatalf("import with unknown asset: err = %v, want validation", err)
	}
	if len(report.RowErrors) != 1 || report.RowErrors[0].Line != 3 {
		t.Fatalf("row errors = %+v", report.RowErrors)
	}
	if findings, _ := f.svc.ListFindings(f.ctx, admin, service.FindingFilter{IncludeResolved: true}); len(findings) != 0 {
		t.Fatalf("nothing should be written on failure, got %d findings", len(findings))
	}

	good := "asset,title,severity,source\nwww.example.test,TLS expired,high,scanner-a\n192.0.2.10,SSH exposed,medium,scanner-a\n"
	report, err = f.svc.ImportFindings(f.ctx, admin, "csv", "scanner-a", strings.NewReader(good), false)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if report.Created != 2 || report.Accepted != 2 {
		t.Fatalf("report = %+v", report)
	}
	notes, err := f.svc.Notifications(f.ctx, admin, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].Kind != domain.NotificationFinding {
		t.Fatalf("expected one notification for the high finding, got %+v", notes)
	}

	report, err = f.svc.ImportFindings(f.ctx, admin, "csv", "scanner-a", strings.NewReader(good), false)
	if err != nil || report.Updated != 2 || report.Created != 0 {
		t.Fatalf("re-import = %+v, %v", report, err)
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	input := "asset,title,severity,source\nwww.example.test,TLS expired,high,scanner-a\n"
	report, err := f.svc.ImportFindings(f.ctx, admin, "csv", "scanner-a", strings.NewReader(input), true)
	if err != nil || !report.DryRun || report.Accepted != 1 {
		t.Fatalf("dry run = %+v, %v", report, err)
	}
	batches, err := f.svc.Batches(f.ctx, admin, "")
	if err != nil || len(batches) != 0 {
		t.Fatalf("dry run created batches: %v %v", batches, err)
	}
}

func TestAcceptingRiskRequiresReason(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	input := "asset,title,severity,source\nwww.example.test,TLS expired,high,scanner-a\n"
	if _, err := f.svc.ImportFindings(f.ctx, admin, "csv", "scanner-a", strings.NewReader(input), false); err != nil {
		t.Fatal(err)
	}
	findings, err := f.svc.ListFindings(f.ctx, admin, service.FindingFilter{})
	if err != nil || len(findings) != 1 {
		t.Fatalf("findings = %v, %v", findings, err)
	}
	id := findings[0].ID
	if _, err := f.svc.SetFindingStatus(f.ctx, admin, id, "accepted", ""); !isErr(err, domain.ErrValidation) {
		t.Fatalf("accept without reason: err = %v", err)
	}
	updated, err := f.svc.SetFindingStatus(f.ctx, admin, id, "accepted", "compensating control in place")
	if err != nil || updated.Status != domain.FindingAccepted {
		t.Fatalf("accept = %+v, %v", updated, err)
	}
	views, err := f.svc.ListFindings(f.ctx, admin, service.FindingFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].Risk.Score != 0 {
		t.Fatalf("accepted risk stays visible at score 0 in the default list, got %+v", views)
	}
	views, err = f.svc.ListFindings(f.ctx, admin, service.FindingFilter{Status: "accepted"})
	if err != nil || len(views) != 1 || views[0].Risk.Score != 0 {
		t.Fatalf("accepted view = %+v, %v", views, err)
	}
}

func TestChangesCompareLatestTwoImports(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	first := "asset,title,severity,source\nwww.example.test,TLS expired,medium,feed\n192.0.2.10,SSH exposed,low,feed\n"
	second := "asset,title,severity,source\nwww.example.test,TLS expired,high,feed\n"
	if _, err := f.svc.ImportFindings(f.ctx, admin, "csv", "feed", strings.NewReader(first), false); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Hour)
	if _, err := f.svc.ImportFindings(f.ctx, admin, "csv", "feed", strings.NewReader(second), false); err != nil {
		t.Fatal(err)
	}
	report, err := f.svc.Changes(f.ctx, admin, "feed")
	if err != nil {
		t.Fatalf("changes: %v", err)
	}
	if len(report.Missing) != 1 || len(report.New) != 0 || len(report.SeverityChanged) != 1 {
		t.Fatalf("report = %+v", report)
	}
	if report.SeverityChanged[0].From != domain.SeverityMedium || report.SeverityChanged[0].To != domain.SeverityHigh {
		t.Fatalf("severity change = %+v", report.SeverityChanged[0])
	}
	if _, err := f.svc.Changes(f.ctx, admin, ""); !isErr(err, domain.ErrValidation) {
		t.Fatalf("missing source: err = %v", err)
	}
}

func TestProbeIsDeniedUntilAllowedAndRejectsOutOfScope(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	var calls atomic.Int32
	dial := func(ctx context.Context, host string, port int, timeout time.Duration) (bool, error) {
		calls.Add(1)
		return port == 443, nil
	}
	req := service.ProbeRequest{Asset: "www.example.test", Ports: []int{80, 443}, Execute: true}
	if _, err := f.svc.Probe(f.ctx, admin, req, dial); !isErr(err, domain.ErrForbidden) {
		t.Fatalf("probe without allowance: err = %v, want forbidden", err)
	}
	if calls.Load() != 0 {
		t.Fatal("dialer must not be called when probing is denied")
	}
	if _, err := f.svc.SetProbeAllowed(f.ctx, admin, "www.example.test", true); err != nil {
		t.Fatal(err)
	}
	result, err := f.svc.Probe(f.ctx, admin, req, dial)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if len(result.Open) != 1 || result.Open[0] != 443 || result.Closed != 1 {
		t.Fatalf("probe result = %+v", result)
	}
	services, err := f.svc.ListServices(f.ctx, admin, "www.example.test")
	if err != nil || len(services) != 1 || services[0].Source != "tcp-connect" {
		t.Fatalf("services = %+v, %v", services, err)
	}
}

func TestProbePlanDoesNotContactTargets(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	if _, err := f.svc.SetProbeAllowed(f.ctx, admin, "192.0.2.10", true); err != nil {
		t.Fatal(err)
	}
	dial := func(ctx context.Context, host string, port int, timeout time.Duration) (bool, error) {
		t.Fatal("plan-only run must not dial")
		return false, nil
	}
	result, err := f.svc.Probe(f.ctx, admin, service.ProbeRequest{Asset: "192.0.2.10", Ports: []int{22, 22, 443}}, dial)
	if err != nil || result.Executed || len(result.Planned) != 2 {
		t.Fatalf("plan = %+v, %v", result, err)
	}
}

func TestProbeBoundsAreEnforced(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	if _, err := f.svc.SetProbeAllowed(f.ctx, admin, "192.0.2.10", true); err != nil {
		t.Fatal(err)
	}
	many := make([]int, 65)
	for i := range many {
		many[i] = i + 1
	}
	cases := map[string]service.ProbeRequest{
		"too many ports": {Asset: "192.0.2.10", Ports: many},
		"port zero":      {Asset: "192.0.2.10", Ports: []int{0}},
		"no ports":       {Asset: "192.0.2.10"},
		"excess timeout": {Asset: "192.0.2.10", Ports: []int{22}, Timeout: time.Minute},
		"excess workers": {Asset: "192.0.2.10", Ports: []int{22}, Concurrency: 99},
	}
	for name, req := range cases {
		if _, err := f.svc.Probe(f.ctx, admin, req, nil); !isErr(err, domain.ErrValidation) {
			t.Errorf("%s: err = %v, want validation", name, err)
		}
	}
}

func TestTicketLifecycleAndOverdue(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	input := "asset,title,severity,source\nwww.example.test,TLS expired,high,scanner-a\n"
	if _, err := f.svc.ImportFindings(f.ctx, admin, "csv", "scanner-a", strings.NewReader(input), false); err != nil {
		t.Fatal(err)
	}
	findings, _ := f.svc.ListFindings(f.ctx, admin, service.FindingFilter{})
	ticket, err := f.svc.CreateTicket(f.ctx, admin, findings[0].ID, "platform-team", "2026-10-30")
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := f.svc.CreateTicket(f.ctx, admin, findings[0].ID, "x", "2026-13-40"); !isErr(err, domain.ErrValidation) {
		t.Fatalf("bad due date: err = %v", err)
	}
	f.now = time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	overdue, err := f.svc.ListTickets(f.ctx, admin, true)
	if err != nil || len(overdue) != 1 {
		t.Fatalf("overdue = %+v, %v", overdue, err)
	}
	created, err := f.svc.RefreshNotifications(f.ctx, admin)
	if err != nil || created < 1 {
		t.Fatalf("refresh created %d, %v", created, err)
	}
	if again, _ := f.svc.RefreshNotifications(f.ctx, admin); again != 0 {
		t.Fatalf("refresh must be idempotent, created %d more", again)
	}
	if _, err := f.svc.SetTicketStatus(f.ctx, admin, ticket.ID, "done"); err != nil {
		t.Fatalf("close ticket: %v", err)
	}
	if overdue, _ := f.svc.ListTickets(f.ctx, admin, true); len(overdue) != 0 {
		t.Fatalf("done tickets must not be overdue, got %d", len(overdue))
	}
}

func TestNmapAndCertificateImportAndExposureScoring(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	nmap := `<nmaprun><host><status state="up"/><address addr="192.0.2.10" addrtype="ipv4"/>
	<ports><port protocol="tcp" portid="22"><state state="open"/><service name="ssh"/></port></ports></host></nmaprun>`
	report, err := f.svc.ImportNmap(f.ctx, admin, strings.NewReader(nmap), false)
	if err != nil || report.Created != 1 {
		t.Fatalf("nmap = %+v, %v", report, err)
	}
	input := "asset,title,severity,source\n192.0.2.10,SSH exposed,medium,scanner-a\n"
	if _, err := f.svc.ImportFindings(f.ctx, admin, "csv", "scanner-a", strings.NewReader(input), false); err != nil {
		t.Fatal(err)
	}
	views, err := f.svc.ListFindings(f.ctx, admin, service.FindingFilter{})
	if err != nil || len(views) != 1 {
		t.Fatalf("views = %+v, %v", views, err)
	}
	if views[0].Risk.Score != 40+2 {
		t.Fatalf("score = %d, want 42 (medium plus one open service)", views[0].Risk.Score)
	}
	search, err := f.svc.Search(f.ctx, admin, "ssh")
	if err != nil || len(search.Services) != 1 || len(search.Findings) != 1 {
		t.Fatalf("search = %+v, %v", search, err)
	}
	if _, err := f.svc.Search(f.ctx, admin, "s"); !isErr(err, domain.ErrValidation) {
		t.Fatalf("short search term: err = %v", err)
	}
}

func TestAuditChainStaysIntact(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	if _, err := f.svc.SetProbeAllowed(f.ctx, admin, "www.example.test", true); err != nil {
		t.Fatal(err)
	}
	entries, verification, err := f.svc.AuditLog(f.ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	if !verification.OK || verification.Chained != len(entries) || len(entries) < 5 {
		t.Fatalf("verification = %+v over %d entries", verification, len(entries))
	}
	if entries[len(entries)-1].Action != "asset.probe_policy" {
		t.Fatalf("last action = %s", entries[len(entries)-1].Action)
	}
}

func TestLastAdministratorCannotBeDemoted(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	if _, err := f.svc.SetUserRole(f.ctx, admin, "root", "viewer"); !isErr(err, domain.ErrValidation) {
		t.Fatalf("demoting last admin: err = %v, want validation", err)
	}
}

func TestDeliverNotificationsWritesOnceAndMarksDelivered(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	input := "asset,title,severity,source\nwww.example.test,Expired TLS,high,scanner-a\n"
	if _, err := f.svc.ImportFindings(f.ctx, admin, "csv", "scanner-a", strings.NewReader(input), false); err != nil {
		t.Fatalf("import: %v", err)
	}
	pending, err := f.svc.PendingNotifications(f.ctx, admin)
	if err != nil || len(pending) == 0 {
		t.Fatalf("pending = %d, %v", len(pending), err)
	}
	var buf strings.Builder
	delivered, err := f.svc.DeliverNotifications(f.ctx, admin, &buf)
	if err != nil || delivered != len(pending) {
		t.Fatalf("deliver = %d, %v; want %d", delivered, err, len(pending))
	}
	if n := strings.Count(buf.String(), "\n"); n != len(pending) {
		t.Fatalf("lines = %d, want %d", n, len(pending))
	}
	again, err := f.svc.DeliverNotifications(f.ctx, admin, &buf)
	if err != nil || again != 0 {
		t.Fatalf("second deliver = %d, %v; want 0", again, err)
	}
}

func TestDeliverNotificationsRequiresOperate(t *testing.T) {
	f := newFixture(t)
	admin := f.admin()
	f.seedInventory(admin)
	viewer := domain.Actor{Name: "vera", Role: domain.RoleViewer}
	if _, err := f.svc.DeliverNotifications(f.ctx, viewer, &strings.Builder{}); !isErr(err, domain.ErrForbidden) {
		t.Fatalf("viewer deliver error = %v, want forbidden", err)
	}
}
