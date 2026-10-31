package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/export"
	"github.com/NexzaDev/argus-asm/internal/outbox"
	"github.com/NexzaDev/argus-asm/internal/service"
)

func (a *App) runFindings(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("findings requires a subcommand: import, list, show, status, batches, or changes")
	}
	rest := args[1:]
	switch args[0] {
	case "import":
		return a.runFindingsImport(ctx, rest)
	case "list":
		return a.runFindingsList(ctx, rest)
	case "show":
		return a.runFindingsShow(ctx, rest)
	case "status":
		return a.runFindingsStatus(ctx, rest)
	case "batches":
		return a.runFindingsBatches(ctx, rest)
	case "changes":
		return a.runFindingsChanges(ctx, rest)
	default:
		return usagef("unknown findings subcommand %q", args[0])
	}
}

func (a *App) runFindingsImport(ctx context.Context, args []string) error {
	flags := newFlagSet("findings import", a.stdout, "findings import --format csv|json|sarif --source NAME <file|-> [--dry-run] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	format := flags.String("format", "csv", "input format: csv, json, or sarif")
	source := flags.String("source", "", "scanner or feed name that produced these findings (required)")
	dryRun := flags.Bool("dry-run", false, "validate and report without writing")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("findings import expects a file path or - for stdin")
	}
	if strings.TrimSpace(*source) == "" {
		return usagef("findings import requires --source")
	}
	reader, closeInput, err := a.openInput(positional[0])
	if err != nil {
		return err
	}
	defer closeInput()
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	report, err := sess.svc.ImportFindings(ctx, sess.actor, *format, *source, reader, *dryRun)
	printRowErrors(a.stderr, report.RowErrors)
	if err != nil {
		return err
	}
	if *dryRun {
		fmt.Fprintf(a.stdout, "dry run: %d row(s) valid; nothing written\n", report.Accepted)
		return nil
	}
	fmt.Fprintf(a.stdout, "imported %d row(s) from %s: %d new, %d updated\n", report.Accepted, report.Source, report.Created, report.Updated)
	return nil
}

func (a *App) runFindingsList(ctx context.Context, args []string) error {
	flags := newFlagSet("findings list", a.stdout, "findings list [--status S] [--severity V] [--asset TEXT] [--all] [--limit N] [--explain] [--json] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	status := flags.String("status", "", "filter by status: open, accepted, or resolved")
	severity := flags.String("severity", "", "filter by severity")
	asset := flags.String("asset", "", "filter by asset text")
	all := flags.Bool("all", false, "include resolved findings")
	limit := flags.Int("limit", 0, "maximum rows (0 means no limit)")
	explain := flags.Bool("explain", false, "show the factors behind each risk score")
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	views, err := sess.svc.ListFindings(ctx, sess.actor, service.FindingFilter{
		Status:          *status,
		Severity:        *severity,
		Asset:           *asset,
		IncludeResolved: *all,
		Limit:           *limit,
	})
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, views)
	}
	if len(views) == 0 {
		fmt.Fprintln(a.stdout, "no findings match; import some with: argus-asm findings import --source NAME <file>")
		return nil
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "ID\tSCORE\tSEVERITY\tSTATUS\tASSET\tTITLE\tSOURCE")
	for _, v := range views {
		fmt.Fprintf(table, "%d\t%d\t%s\t%s\t%s\t%s\t%s\n", v.ID, v.Risk.Score, v.Severity, v.Status, v.AssetValue, v.Title, v.Source)
		if *explain {
			for _, factor := range v.Risk.Factors {
				fmt.Fprintf(table, "\t\t\t\t  - %s\t\t\n", factor)
			}
		}
	}
	return table.Flush()
}

func parseID(name, raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, usagef("%s must be a positive integer, got %q", name, raw)
	}
	return id, nil
}

func (a *App) runFindingsShow(ctx context.Context, args []string) error {
	flags := newFlagSet("findings show", a.stdout, "findings show <id> [--json] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("findings show expects one finding id")
	}
	id, err := parseID("finding id", positional[0])
	if err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	detail, err := sess.svc.ShowFinding(ctx, sess.actor, id)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, detail)
	}
	f := detail.Finding
	fmt.Fprintf(a.stdout, "finding %d: %s\n", f.ID, f.Title)
	fmt.Fprintf(a.stdout, "  asset:    %s\n", f.AssetValue)
	fmt.Fprintf(a.stdout, "  severity: %s   status: %s   risk score: %d\n", f.Severity, f.Status, f.Risk.Score)
	fmt.Fprintf(a.stdout, "  source:   %s\n", f.Source)
	fmt.Fprintf(a.stdout, "  first seen: %s   last seen: %s\n", formatTimestamp(f.FirstSeen), formatTimestamp(f.LastSeen))
	if f.StatusReason != "" {
		fmt.Fprintf(a.stdout, "  status reason: %s\n", f.StatusReason)
	}
	fmt.Fprintln(a.stdout, "  risk factors:")
	for _, factor := range f.Risk.Factors {
		fmt.Fprintf(a.stdout, "    - %s\n", factor)
	}
	fmt.Fprintf(a.stdout, "  tickets: %d\n", len(detail.Tickets))
	for _, t := range detail.Tickets {
		fmt.Fprintf(a.stdout, "    #%d %s owner=%s due=%s\n", t.ID, t.Status, t.Owner, t.DueDate.Format("2006-01-02"))
	}
	return nil
}

func (a *App) runFindingsStatus(ctx context.Context, args []string) error {
	flags := newFlagSet("findings status", a.stdout, "findings status <id> --set open|accepted|resolved [--reason TEXT] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	target := flags.String("set", "", "new status: open, accepted, or resolved (required)")
	reason := flags.String("reason", "", "reason; required when accepting a risk")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("findings status expects one finding id")
	}
	id, err := parseID("finding id", positional[0])
	if err != nil {
		return err
	}
	if *target == "" {
		return usagef("findings status requires --set")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	finding, err := sess.svc.SetFindingStatus(ctx, sess.actor, id, *target, *reason)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "finding %d is now %s\n", finding.ID, finding.Status)
	return nil
}

func (a *App) runFindingsBatches(ctx context.Context, args []string) error {
	flags := newFlagSet("findings batches", a.stdout, "findings batches [--source NAME] [--db PATH] [--actor NAME] [--json]")
	common := addCommon(flags)
	source := flags.String("source", "", "limit to one source")
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	batches, err := sess.svc.Batches(ctx, sess.actor, *source)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, batches)
	}
	if len(batches) == 0 {
		fmt.Fprintln(a.stdout, "no import batches recorded")
		return nil
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "BATCH\tIMPORTED\tFORMAT\tSOURCE\tROWS\tNEW\tUPDATED")
	for _, b := range batches {
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%d\t%d\t%d\n", b.ID, formatTimestamp(b.CreatedAt), b.Format, b.Source, b.Accepted, b.Created, b.Updated)
	}
	return table.Flush()
}

func (a *App) runFindingsChanges(ctx context.Context, args []string) error {
	flags := newFlagSet("findings changes", a.stdout, "findings changes --source NAME [--db PATH] [--actor NAME] [--json]")
	common := addCommon(flags)
	source := flags.String("source", "", "source to compare (required)")
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	report, err := sess.svc.Changes(ctx, sess.actor, *source)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, report)
	}
	fmt.Fprintf(a.stdout, "changes for %s between batch %d and batch %d\n", report.Source, report.Previous.ID, report.Latest.ID)
	fmt.Fprintf(a.stdout, "  new: %d   missing: %d   severity changed: %d\n", len(report.New), len(report.Missing), len(report.SeverityChanged))
	for _, o := range report.New {
		fmt.Fprintf(a.stdout, "  + [%s] %s: %s\n", o.Severity, o.AssetValue, o.Title)
	}
	for _, o := range report.Missing {
		fmt.Fprintf(a.stdout, "  - [%s] %s: %s\n", o.Severity, o.AssetValue, o.Title)
	}
	for _, c := range report.SeverityChanged {
		fmt.Fprintf(a.stdout, "  ~ %s: %s %s -> %s\n", c.AssetValue, c.Title, c.From, c.To)
	}
	return nil
}

func (a *App) runTicket(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("ticket requires a subcommand: create, list, or status")
	}
	rest := args[1:]
	switch args[0] {
	case "create":
		return a.runTicketCreate(ctx, rest)
	case "list":
		return a.runTicketList(ctx, rest)
	case "status":
		return a.runTicketStatus(ctx, rest)
	default:
		return usagef("unknown ticket subcommand %q", args[0])
	}
}

func (a *App) runTicketCreate(ctx context.Context, args []string) error {
	flags := newFlagSet("ticket create", a.stdout, "ticket create --finding ID --owner NAME --due YYYY-MM-DD [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	finding := flags.String("finding", "", "finding id (required)")
	owner := flags.String("owner", "", "ticket owner (required)")
	due := flags.String("due", "", "due date YYYY-MM-DD (required)")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	id, err := parseID("--finding", *finding)
	if err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	ticket, err := sess.svc.CreateTicket(ctx, sess.actor, id, *owner, *due)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "created ticket %d for finding %d (owner %s, due %s)\n", ticket.ID, ticket.FindingID, ticket.Owner, ticket.DueDate.Format("2006-01-02"))
	return nil
}

func (a *App) runTicketList(ctx context.Context, args []string) error {
	flags := newFlagSet("ticket list", a.stdout, "ticket list [--overdue] [--db PATH] [--actor NAME] [--json]")
	common := addCommon(flags)
	overdue := flags.Bool("overdue", false, "only tickets past their due date that are not done")
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	tickets, err := sess.svc.ListTickets(ctx, sess.actor, *overdue)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, tickets)
	}
	if len(tickets) == 0 {
		fmt.Fprintln(a.stdout, "no tickets match")
		return nil
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "ID\tSTATUS\tDUE\tOWNER\tASSET\tFINDING")
	for _, t := range tickets {
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\t%s\n", t.ID, t.Status, t.DueDate.Format("2006-01-02"), t.Owner, t.AssetValue, t.FindingTitle)
	}
	return table.Flush()
}

func (a *App) runTicketStatus(ctx context.Context, args []string) error {
	flags := newFlagSet("ticket status", a.stdout, "ticket status <id> --set open|in_progress|done [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	target := flags.String("set", "", "new status (required)")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("ticket status expects one ticket id")
	}
	id, err := parseID("ticket id", positional[0])
	if err != nil {
		return err
	}
	if *target == "" {
		return usagef("ticket status requires --set")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	ticket, err := sess.svc.SetTicketStatus(ctx, sess.actor, id, *target)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "ticket %d is now %s\n", ticket.ID, ticket.Status)
	return nil
}

func (a *App) runSearch(ctx context.Context, args []string) error {
	flags := newFlagSet("search", a.stdout, "search <term> [--db PATH] [--actor NAME] [--json]")
	common := addCommon(flags)
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("search expects exactly one term")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	result, err := sess.svc.Search(ctx, sess.actor, positional[0])
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, result)
	}
	fmt.Fprintf(a.stdout, "assets: %d   findings: %d   services: %d\n", len(result.Assets), len(result.Findings), len(result.Services))
	for _, asset := range result.Assets {
		fmt.Fprintf(a.stdout, "  asset %s %s\n", asset.Kind, asset.Value)
	}
	for _, f := range result.Findings {
		fmt.Fprintf(a.stdout, "  finding %d [%s] %s: %s\n", f.ID, f.Severity, f.AssetValue, f.Title)
	}
	for _, svc := range result.Services {
		fmt.Fprintf(a.stdout, "  service %s %d/%s %s\n", svc.AssetValue, svc.Port, svc.Protocol, svc.Name)
	}
	return nil
}

func (a *App) runNotify(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("notify requires a subcommand: list, ack, refresh, or deliver")
	}
	rest := args[1:]
	switch args[0] {
	case "list":
		return a.runNotifyList(ctx, rest)
	case "ack":
		return a.runNotifyAck(ctx, rest)
	case "refresh":
		return a.runNotifyRefresh(ctx, rest)
	case "deliver":
		return a.runNotifyDeliver(ctx, rest)
	default:
		return usagef("unknown notify subcommand %q", args[0])
	}
}

func (a *App) runNotifyList(ctx context.Context, args []string) error {
	flags := newFlagSet("notify list", a.stdout, "notify list [--all] [--db PATH] [--actor NAME] [--json]")
	common := addCommon(flags)
	all := flags.Bool("all", false, "include acknowledged notifications")
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	notes, err := sess.svc.Notifications(ctx, sess.actor, *all)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, notes)
	}
	if len(notes) == 0 {
		fmt.Fprintln(a.stdout, "no notifications")
		return nil
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "ID\tCREATED\tKIND\tACKED\tMESSAGE")
	for _, n := range notes {
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\n", n.ID, formatTimestamp(n.CreatedAt), n.Kind, formatOptionalTimestamp(n.AcknowledgedAt), n.Message)
	}
	return table.Flush()
}

func (a *App) runNotifyAck(ctx context.Context, args []string) error {
	flags := newFlagSet("notify ack", a.stdout, "notify ack <id> [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("notify ack expects one notification id")
	}
	id, err := parseID("notification id", positional[0])
	if err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	if err := sess.svc.Acknowledge(ctx, sess.actor, id); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "acknowledged notification %d\n", id)
	return nil
}

func (a *App) runNotifyRefresh(ctx context.Context, args []string) error {
	flags := newFlagSet("notify refresh", a.stdout, "notify refresh [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	created, err := sess.svc.RefreshNotifications(ctx, sess.actor)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "created %d new notification(s)\n", created)
	return nil
}

func (a *App) runNotifyDeliver(ctx context.Context, args []string) error {
	flags := newFlagSet("notify deliver", a.stdout, "notify deliver --to FILE [--dry-run] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	to := flags.String("to", "", "local outbox file to append JSON lines to (created with mode 0600)")
	dryRun := flags.Bool("dry-run", false, "count undelivered notifications without writing")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	if strings.TrimSpace(*to) == "" {
		return usagef("notify deliver requires --to FILE")
	}
	if err := outbox.Check(*to); err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	pending, err := sess.svc.PendingNotifications(ctx, sess.actor)
	if err != nil {
		return err
	}
	if *dryRun {
		fmt.Fprintf(a.stdout, "dry run: %d undelivered notification(s); nothing written\n", len(pending))
		return nil
	}
	if len(pending) == 0 {
		fmt.Fprintf(a.stdout, "delivered 0 notification(s) to %s\n", *to)
		return nil
	}
	file, err := outbox.Open(*to)
	if err != nil {
		return err
	}
	defer file.Close()
	delivered, err := sess.svc.DeliverNotifications(ctx, sess.actor, file)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close outbox: %w", err)
	}
	fmt.Fprintf(a.stdout, "delivered %d notification(s) to %s\n", delivered, *to)
	return nil
}

func (a *App) runUser(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("user requires a subcommand: add, list, or role")
	}
	rest := args[1:]
	switch args[0] {
	case "add":
		return a.runUserAdd(ctx, rest)
	case "list":
		return a.runUserList(ctx, rest)
	case "role":
		return a.runUserRole(ctx, rest)
	default:
		return usagef("unknown user subcommand %q", args[0])
	}
}

func (a *App) runUserAdd(ctx context.Context, args []string) error {
	flags := newFlagSet("user add", a.stdout, "user add <name> --role viewer|analyst|admin [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	role := flags.String("role", "", "viewer, analyst, or admin (required)")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("user add expects one user name")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	user, err := sess.svc.CreateUser(ctx, sess.actor, positional[0], *role)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "added user %s with role %s\n", user.Name, user.Role)
	return nil
}

func (a *App) runUserList(ctx context.Context, args []string) error {
	flags := newFlagSet("user list", a.stdout, "user list [--db PATH] [--actor NAME] [--json]")
	common := addCommon(flags)
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	users, err := sess.svc.ListUsers(ctx, sess.actor)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, users)
	}
	if len(users) == 0 {
		fmt.Fprintf(a.stdout, "no users defined; the bootstrap actor %q has administrator rights until one is added\n", domain.BootstrapActorName)
		return nil
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "NAME\tROLE\tCREATED")
	for _, u := range users {
		fmt.Fprintf(table, "%s\t%s\t%s\n", u.Name, u.Role, formatTimestamp(u.CreatedAt))
	}
	return table.Flush()
}

func (a *App) runUserRole(ctx context.Context, args []string) error {
	flags := newFlagSet("user role", a.stdout, "user role <name> --role viewer|analyst|admin [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	role := flags.String("role", "", "new role (required)")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("user role expects one user name")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	user, err := sess.svc.SetUserRole(ctx, sess.actor, positional[0], *role)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "user %s now has role %s\n", user.Name, user.Role)
	return nil
}

func (a *App) runAudit(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("audit requires a subcommand: list or verify")
	}
	flags := newFlagSet("audit "+args[0], a.stdout, "audit "+args[0]+" [--db PATH] [--actor NAME] [--json]")
	common := addCommon(flags)
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	if args[0] != "list" && args[0] != "verify" {
		return usagef("unknown audit subcommand %q", args[0])
	}
	if _, err := parseArgs(flags, args[1:]); err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	entries, verification, err := sess.svc.AuditLog(ctx, sess.actor)
	if err != nil {
		return err
	}
	if args[0] == "verify" {
		if *jsonOut {
			return writeJSON(a.stdout, verification)
		}
		if !verification.OK {
			return fmt.Errorf("audit chain is broken: %s", verification.Problem)
		}
		fmt.Fprintf(a.stdout, "audit chain intact: %d chained entries verified, %d legacy entries without hashes\n",
			verification.Chained, verification.Legacy)
		return nil
	}
	if *jsonOut {
		return writeJSON(a.stdout, entries)
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "ID\tAT\tACTOR\tACTION\tDETAIL")
	for _, e := range entries {
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\n", e.ID, formatTimestamp(e.At), e.Actor, e.Action, e.Detail)
	}
	return table.Flush()
}

func (a *App) runExport(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("export requires a dataset: findings or assets")
	}
	dataset := args[0]
	if dataset != "findings" && dataset != "assets" {
		return usagef("unknown export dataset %q", dataset)
	}
	flags := newFlagSet("export "+dataset, a.stdout, "export "+dataset+" --format csv|json [--all] [--output FILE] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	format := flags.String("format", "csv", "csv or json")
	all := flags.Bool("all", false, "include resolved findings")
	output := flags.String("output", "", "write to FILE instead of stdout (created with mode 0600)")
	if _, err := parseArgs(flags, args[1:]); err != nil {
		return err
	}
	if *format != "csv" && *format != "json" {
		return usagef("--format must be csv or json")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	var out io.Writer = a.stdout
	if *output != "" {
		file, err := os.OpenFile(*output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("create export file: %w", err)
		}
		defer file.Close()
		out = file
	}
	if dataset == "findings" {
		views, err := sess.svc.ExportFindings(ctx, sess.actor, *all)
		if err != nil {
			return err
		}
		if *format == "json" {
			return writeJSON(out, views)
		}
		return export.WriteFindingsCSV(out, views)
	}
	assets, err := sess.svc.ExportAssets(ctx, sess.actor)
	if err != nil {
		return err
	}
	if *format == "json" {
		return writeJSON(out, assets)
	}
	return export.WriteAssetsCSV(out, assets)
}
