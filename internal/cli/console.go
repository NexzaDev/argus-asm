package cli

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"unicode"
)

var consoleCommandsWithCommonFlags = map[string]bool{
	"init": true, "diagnostics": true, "scope": true, "asset": true, "service": true,
	"cert": true, "probe": true, "findings": true, "ticket": true, "search": true,
	"notify": true, "user": true, "audit": true, "export": true,
}

func (a *App) runConsole(ctx context.Context, args []string) error {
	if a.inConsole {
		return usagef("the console cannot be started inside the console")
	}
	flags := newFlagSet("console", a.stdout, "console [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	child := *a
	child.inConsole = true
	scanner := bufio.NewScanner(a.in)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)
	fmt.Fprintln(a.stdout, "argus-asm console: each line runs one CLI command. Type 'help' for commands, 'exit' to quit.")
	fmt.Fprintln(a.stdout, "Inside the console, read input from files; '-' (stdin) is not available.")
	for {
		fmt.Fprint(a.stdout, "argus-asm> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}
		tokens, err := splitLine(line)
		if err != nil {
			fmt.Fprintf(a.stderr, "error: %v\n", err)
			continue
		}
		if len(tokens) == 0 {
			continue
		}
		if tokens[0] == "help" || tokens[0] == "-h" || tokens[0] == "--help" {
			printUsage(a.stdout)
			continue
		}
		tokens = withConsoleDefaults(tokens, *common.db, *common.actor)
		_ = a.report(child.dispatch(ctx, tokens[0], tokens[1:]))
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read console input: %w", err)
	}
	return nil
}

func withConsoleDefaults(tokens []string, db, actor string) []string {
	if len(tokens) == 0 || !consoleCommandsWithCommonFlags[tokens[0]] {
		return tokens
	}
	out := append([]string{}, tokens...)
	if db != "" && !hasFlag(tokens, "--db") && !hasFlag(tokens, "-db") {
		out = append(out, "--db", db)
	}
	if actor != "" && !hasFlag(tokens, "--actor") && !hasFlag(tokens, "-actor") {
		out = append(out, "--actor", actor)
	}
	return out
}

func hasFlag(tokens []string, name string) bool {
	for _, token := range tokens {
		if token == name || strings.HasPrefix(token, name+"=") {
			return true
		}
	}
	return false
}

func splitLine(line string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inToken := false
	var quote rune
	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
			inToken = true
		case unicode.IsSpace(r):
			if inToken {
				tokens = append(tokens, current.String())
				current.Reset()
				inToken = false
			}
		default:
			current.WriteRune(r)
			inToken = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated %c quote", quote)
	}
	if inToken {
		tokens = append(tokens, current.String())
	}
	return tokens, nil
}
