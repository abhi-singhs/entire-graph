package cli

import (
	"flag"
	"fmt"
	"github.com/entireio/entire-graph/internal/agentsetup"
)

const agentGuide = agentsetup.GraphGuide

func graphAgentCommand(opts Options, args []string, install bool) error {
	fs := flag.NewFlagSet("agent instructions", flag.ContinueOnError)
	fs.SetOutput(opts.Stderr)
	repo := fs.String("repo", "", "Project root (default: host repository or nearest repository)")
	strict := fs.Bool("strict", false, "Use strict guidance (saved by init-agents; preview only for agent-guide)")
	normal := fs.Bool("normal", false, "Use normal guidance (saved by init-agents; preview only for agent-guide)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	mode, err := agentsetup.SelectMode(*strict, *normal)
	if err != nil {
		return err
	}
	root, err := agentsetup.Context(*repo, opts.Env.RepoRoot)
	if err != nil {
		return err
	}
	render := func() (string, error) { return agentsetup.Preview(root, "graph", agentsetup.Options{Mode: mode}) }
	if install {
		if root == "" {
			return fmt.Errorf("outside a repository; supply --repo")
		}
		return agentsetup.Install(root, render, opts.Stdout)
	}
	guide, err := render()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(opts.Stdout, guide)
	return err
}
func runAgentGuide(opts Options, args []string) error { return graphAgentCommand(opts, args, false) }
func runInitAgents(opts Options, args []string) error { return graphAgentCommand(opts, args, true) }
