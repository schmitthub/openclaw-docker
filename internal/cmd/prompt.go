package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func confirmWrite(cmd *cobra.Command, dangerousInline bool, target string) error {
	if dangerousInline {
		return nil
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Defensive warning: write operation target: %s\n", target)
	fmt.Fprint(cmd.ErrOrStderr(), "Proceed with this write? [y/N]: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil && len(input) == 0 {
		return fmt.Errorf("write aborted for %s (no confirmation provided; use --dangerous-inline to skip prompts)", target)
	}

	answer := strings.ToLower(strings.TrimSpace(input))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("write aborted for %s", target)
	}

	return nil
}
