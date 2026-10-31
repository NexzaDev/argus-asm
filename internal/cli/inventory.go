package cli

import (
	"context"
	"fmt"
)

func (a *App) runScope(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("scope requires a subcommand: add or list")
	}
	switch args[0] {
	case "add":
		return a.runScopeAdd(ctx, args[1:])
	case "list":
		return a.runScopeList(ctx, args[1:])
	default:
		return usagef("unknown scope subcommand %q", args[0])
	}
}

func (a *App) runScopeAdd(ctx context.Context, args []string) error {
	flags := newFlagSet("scope add", a.stdout, "scope add <domain|cidr> [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("scope add expects exactly one domain or CIDR value")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	scope, err := sess.svc.AddScope(ctx, sess.actor, positional[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "added %s scope %s\n", scope.Kind, scope.Value)
	return nil
}

func (a *App) runScopeList(ctx context.Context, args []string) error {
	flags := newFlagSet("scope list", a.stdout, "scope list [--db PATH] [--actor NAME] [--json]")
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
	scopes, err := sess.svc.ListScopes(ctx, sess.actor)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, scopes)
	}
	if len(scopes) == 0 {
		fmt.Fprintln(a.stdout, "no scopes defined; add one with: argus-asm scope add <domain|cidr>")
		return nil
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "KIND\tVALUE\tCREATED")
	for _, scope := range scopes {
		fmt.Fprintf(table, "%s\t%s\t%s\n", scope.Kind, scope.Value, formatTimestamp(scope.CreatedAt))
	}
	return table.Flush()
}

func (a *App) runAsset(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("asset requires a subcommand: add, list, show, remove, owner, link, links, or probe")
	}
	rest := args[1:]
	switch args[0] {
	case "add":
		return a.runAssetAdd(ctx, rest)
	case "list":
		return a.runAssetList(ctx, rest)
	case "show":
		return a.runAssetShow(ctx, rest)
	case "remove":
		return a.runAssetRemove(ctx, rest)
	case "owner":
		return a.runAssetOwner(ctx, rest)
	case "link":
		return a.runAssetLink(ctx, rest)
	case "links":
		return a.runAssetLinks(ctx, rest)
	case "probe":
		return a.runAssetProbe(ctx, rest)
	default:
		return usagef("unknown asset subcommand %q", args[0])
	}
}

func (a *App) runAssetAdd(ctx context.Context, args []string) error {
	flags := newFlagSet("asset add", a.stdout, "asset add <domain|ip> [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("asset add expects exactly one domain or IP value")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	asset, err := sess.svc.AddAsset(ctx, sess.actor, positional[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "added %s asset %s\n", asset.Kind, asset.Value)
	return nil
}

func (a *App) runAssetList(ctx context.Context, args []string) error {
	flags := newFlagSet("asset list", a.stdout, "asset list [--db PATH] [--actor NAME] [--json]")
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
	assets, err := sess.svc.ListAssets(ctx, sess.actor)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, assets)
	}
	if len(assets) == 0 {
		fmt.Fprintln(a.stdout, "no assets registered; add one with: argus-asm asset add <domain|ip>")
		return nil
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "KIND\tVALUE\tOWNER\tVERIFIED_BY\tPROBE\tCREATED")
	for _, asset := range assets {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\n", asset.Kind, asset.Value, dashIfEmpty(asset.OwnerName),
			dashIfEmpty(asset.VerifiedBy), probeLabel(asset.ProbeAllowed), formatTimestamp(asset.CreatedAt))
	}
	return table.Flush()
}

func dashIfEmpty(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func probeLabel(allowed bool) string {
	if allowed {
		return "allowed"
	}
	return "denied"
}

func (a *App) runAssetShow(ctx context.Context, args []string) error {
	flags := newFlagSet("asset show", a.stdout, "asset show <domain|ip> [--db PATH] [--actor NAME] [--json]")
	common := addCommon(flags)
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("asset show expects exactly one asset value")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	asset, err := sess.svc.ShowAsset(ctx, sess.actor, positional[0])
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, asset)
	}
	fmt.Fprintf(a.stdout, "kind:        %s\n", asset.Kind)
	fmt.Fprintf(a.stdout, "value:       %s\n", asset.Value)
	fmt.Fprintf(a.stdout, "owner:       %s\n", dashIfEmpty(asset.OwnerName))
	fmt.Fprintf(a.stdout, "verified by: %s\n", dashIfEmpty(asset.VerifiedBy))
	fmt.Fprintf(a.stdout, "verified at: %s\n", formatOptionalTimestamp(asset.VerifiedAt))
	fmt.Fprintf(a.stdout, "probing:     %s\n", probeLabel(asset.ProbeAllowed))
	fmt.Fprintf(a.stdout, "created:     %s\n", formatTimestamp(asset.CreatedAt))
	return nil
}

func (a *App) runAssetRemove(ctx context.Context, args []string) error {
	flags := newFlagSet("asset remove", a.stdout, "asset remove <domain|ip> [--dry-run] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	dryRun := flags.Bool("dry-run", false, "report what would be deleted without deleting")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("asset remove expects exactly one asset value")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	asset, impact, err := sess.svc.RemoveAsset(ctx, sess.actor, positional[0], *dryRun)
	if err != nil {
		return err
	}
	verb := "removed"
	if *dryRun {
		verb = "would remove"
	}
	fmt.Fprintf(a.stdout, "%s %s %s\n", verb, asset.Kind, asset.Value)
	fmt.Fprintf(a.stdout, "  findings: %d  services: %d  certificates: %d  relations: %d  tickets: %d\n",
		impact.Findings, impact.Services, impact.Certificates, impact.Relations, impact.Tickets)
	return nil
}

func (a *App) runAssetOwner(ctx context.Context, args []string) error {
	flags := newFlagSet("asset owner", a.stdout, "asset owner <domain|ip> --owner NAME [--verified-by NAME] [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	owner := flags.String("owner", "", "owning team or person (required)")
	verifiedBy := flags.String("verified-by", "", "person who verified ownership (sets the verification time)")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("asset owner expects exactly one asset value")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	asset, err := sess.svc.SetOwnership(ctx, sess.actor, positional[0], *owner, *verifiedBy)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "owner of %s is %s (verified by %s)\n", asset.Value, asset.OwnerName, dashIfEmpty(asset.VerifiedBy))
	return nil
}

func (a *App) runAssetLink(ctx context.Context, args []string) error {
	flags := newFlagSet("asset link", a.stdout, "asset link <from> <to> --kind resolves_to|hosts|related_to [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	kind := flags.String("kind", "", "relation kind (required)")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 2 {
		return usagef("asset link expects a source and a target asset")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	rel, err := sess.svc.Link(ctx, sess.actor, positional[0], positional[1], *kind)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "linked %s %s %s\n", rel.FromValue, rel.Kind, rel.ToValue)
	return nil
}

func (a *App) runAssetLinks(ctx context.Context, args []string) error {
	flags := newFlagSet("asset links", a.stdout, "asset links <domain|ip> [--db PATH] [--actor NAME] [--json]")
	common := addCommon(flags)
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	positional, err := parseArgs(flags, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("asset links expects exactly one asset value")
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	rels, err := sess.svc.Relations(ctx, sess.actor, positional[0])
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a.stdout, rels)
	}
	if len(rels) == 0 {
		fmt.Fprintln(a.stdout, "no relations recorded for this asset")
		return nil
	}
	table := newTable(a.stdout)
	fmt.Fprintln(table, "FROM\tKIND\tTO")
	for _, rel := range rels {
		fmt.Fprintf(table, "%s\t%s\t%s\n", rel.FromValue, rel.Kind, rel.ToValue)
	}
	return table.Flush()
}

func (a *App) runAssetProbe(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("asset probe requires a subcommand: allow or deny")
	}
	var allowed bool
	switch args[0] {
	case "allow":
		allowed = true
	case "deny":
		allowed = false
	default:
		return usagef("unknown asset probe subcommand %q", args[0])
	}
	flags := newFlagSet("asset probe "+args[0], a.stdout, "asset probe "+args[0]+" <domain|ip> [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	positional, err := parseArgs(flags, args[1:])
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usagef("asset probe %s expects exactly one asset value", args[0])
	}
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	asset, err := sess.svc.SetProbeAllowed(ctx, sess.actor, positional[0], allowed)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "active probing of %s is now %s\n", asset.Value, probeLabel(asset.ProbeAllowed))
	return nil
}
