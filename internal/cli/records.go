package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/NexzaDev/argus-asm/internal/importer"
	"github.com/NexzaDev/argus-asm/internal/service"
)

func (a *App) runService(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("service requires a subcommand: add, list, or import-nmap")
	}
	switch args[0] {
	case "add":
		return a.runServiceAdd(ctx, args[1:])
	case "list":
		return a.runServiceList(ctx, args[1:])
	case "import-nmap":
		return a.runServiceImportNmap(ctx, args[1:])
	default:
		return usagef("unknown service subcommand %q", args[0])
	}
}

func (a *App) runServiceAdd(ctx context.Context, args []string) error {
	flags := newFlagSet("service add", a.stdout, "service add <domain|ip> <port[/tcp|udp]> [--name NAME] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	name := flags.String("name", "", "service name, for example https")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 2 {
		return usagef("service add expects an asset and a port")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	rec, created, err := sess.svc.AddService(ctx, sess.actor, positional[0], positional[1], *name)
	if err != nil {
		return err
	}
	verb := "updated"
	if created {
		verb = "added"
	}
	fmt.Fprintf(a.stdout, "%s service %s %d/%s\n", verb, rec.AssetValue, rec.Port, rec.Protocol)
	return nil
}

func (a *App) runServiceList(ctx context.Context, args []string) error {
	flags := newFlagSet("service list", a.stdout, "service list [--asset ASSET] [--db PATH] [--actor NAME] [--json]")
	common := addCommon(flags)
	asset := flags.String("asset", "", "limit to one asset")
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	records, err := sess.svc.ListServices(ctx, sess.actor, *asset)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, records)
	}
	if len(records) == 0 {
		fmt.Fprintln(a.stdout, "no services recorded")
		return nil
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "ASSET\tPORT\tPROTO\tNAME\tSOURCE\tLAST_SEEN")
	for _, rec := range records {
		fmt.Fprintf(table, "%s\t%d\t%s\t%s\t%s\t%s\n", rec.AssetValue, rec.Port, rec.Protocol,
			dashIfEmpty(rec.Name), rec.Source, formatTimestamp(rec.LastSeen))
	}
	return table.Flush()
}

func (a *App) runServiceImportNmap(ctx context.Context, args []string) error {
	flags := newFlagSet("service import-nmap", a.stdout, "service import-nmap <file|-> [--dry-run] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	dryRun := flags.Bool("dry-run", false, "validate and report without writing")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("service import-nmap expects a file path or - for stdin")
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
	report, err := sess.svc.ImportNmap(ctx, sess.actor, reader, *dryRun)
	printRowErrors(a.stderr, report.RowErrors)
	if err != nil {
		return err
	}
	if *dryRun {
		fmt.Fprintf(a.stdout, "dry run: %d service observation(s) valid; nothing written\n", report.Accepted)
		return nil
	}
	fmt.Fprintf(a.stdout, "imported nmap services: %d observed, %d new, %d updated\n", report.Accepted, report.Created, report.Updated)
	return nil
}

func (a *App) openInput(path string) (io.Reader, func(), error) {
	if path == "-" {
		if a.inConsole {
			return nil, nil, usagef("'-' (stdin) is not available inside the console; pass a file path")
		}
		return a.in, func() {}, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open input: %w", err)
	}
	return file, func() { _ = file.Close() }, nil
}

func (a *App) runCert(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("cert requires a subcommand: import or list")
	}
	switch args[0] {
	case "import":
		return a.runCertImport(ctx, args[1:])
	case "list":
		return a.runCertList(ctx, args[1:])
	default:
		return usagef("unknown cert subcommand %q", args[0])
	}
}

func (a *App) runCertImport(ctx context.Context, args []string) error {
	flags := newFlagSet("cert import", a.stdout, "cert import <domain|ip> <pem-file> [--dry-run] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	dryRun := flags.Bool("dry-run", false, "validate and report without writing")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 2 {
		return usagef("cert import expects an asset and a PEM file")
	}
	data, err := os.ReadFile(positional[1])
	if err != nil {
		return fmt.Errorf("read certificate file: %w", err)
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	report, err := sess.svc.ImportCertificates(ctx, sess.actor, positional[0], data, *dryRun)
	if err != nil {
		return err
	}
	if *dryRun {
		fmt.Fprintf(a.stdout, "dry run: %d certificate(s) valid for %s; nothing written\n", report.Accepted, report.Asset)
		return nil
	}
	fmt.Fprintf(a.stdout, "imported certificates for %s: %d new, %d already known, %d expiring within 30 days\n",
		report.Asset, report.Created, report.Duplicates, report.Expiring)
	return nil
}

func (a *App) runCertList(ctx context.Context, args []string) error {
	flags := newFlagSet("cert list", a.stdout, "cert list [--db PATH] [--actor NAME] [--json]")
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
	certs, err := sess.svc.ListCertificates(ctx, sess.actor)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, certs)
	}
	if len(certs) == 0 {
		fmt.Fprintln(a.stdout, "no certificates recorded")
		return nil
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "ASSET\tNOT_AFTER\tSUBJECT\tISSUER\tSHA256")
	for _, c := range certs {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", c.AssetValue, c.NotAfter.Format("2006-01-02"),
			c.Subject, c.Issuer, c.Fingerprint[:16])
	}
	return table.Flush()
}

func (a *App) runProbe(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "run" {
		return usagef("probe requires the subcommand run: probe run <asset> --ports LIST")
	}
	flags := newFlagSet("probe run", a.stdout, "probe run <domain|ip> --ports 22,443 [--execute] [--timeout 2s] [--concurrency 4] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	portList := flags.String("ports", "", "comma-separated TCP ports, at most 64")
	execute := flags.Bool("execute", false, "actually open connections (default: plan only)")
	timeout := flags.Duration("timeout", 2*time.Second, "per-connection timeout (100ms to 10s)")
	concurrency := flags.Int("concurrency", 4, "parallel connections (1 to 16)")
	positional, err := parseArgs(flags, args[1:])
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("probe run expects exactly one asset value")
	}
	ports, err := parsePortList(*portList)
	if err != nil {
		return err
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	result, err := sess.svc.Probe(ctx, sess.actor, service.ProbeRequest{
		Asset:       positional[0],
		Ports:       ports,
		Execute:     *execute,
		Timeout:     *timeout,
		Concurrency: *concurrency,
	}, a.dial)
	if err != nil {
		return err
	}
	if !result.Executed {
		fmt.Fprintf(a.stdout, "plan for %s: %d TCP port(s) %s; nothing was contacted (add --execute to run)\n",
			result.Asset, len(result.Planned), joinInts(result.Planned))
		return nil
	}
	fmt.Fprintf(a.stdout, "probed %s: %d open, %d closed or filtered\n", result.Asset, len(result.Open), result.Closed)
	for _, port := range result.Open {
		fmt.Fprintf(a.stdout, "  open %d/tcp\n", port)
	}
	return nil
}

func parsePortList(raw string) ([]int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, usagef("--ports is required, for example --ports 22,443")
	}
	var ports []int
	for _, part := range strings.Split(raw, ",") {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, usagef("port %q is not a number", part)
		}
		ports = append(ports, value)
	}
	return ports, nil
}

func joinInts(values []int) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

func printRowErrors(w io.Writer, rowErrors []importer.RowError) {
	for _, rowErr := range rowErrors {
		fmt.Fprintf(w, "line %d: %s\n", rowErr.Line, rowErr.Message)
	}
}
