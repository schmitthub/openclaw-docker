package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func confirmWrite(cmd *cobra.Command, dangerousInline bool, target string) error {
	if dangerousInline {
		return nil
	}

	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("check write target %s: %w", target, err)
	}
	if info.IsDir() {
		return fmt.Errorf("write target is a directory: %s", target)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Warning! this operation will overwrite: %s\n", target)
	fmt.Fprint(cmd.ErrOrStderr(), "Continue? [y/N]: ")

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
