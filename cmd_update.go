package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func pkr_make_cmd_update() *commander.Command {
	cmd := &commander.Command{
		Run:       pkr_run_cmd_update,
		UsageLine: "update [options]",
		Short:     "update RPMs from the yum repository",
		Long: `
update updates RPMs from the yum repository.

ex:
 $ pkr update
`,
		Flag: *flag.NewFlagSet("pkr-update", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	cmd.Flag.String("type", "lhcb", "config type (lhcb|atlas)")
	return cmd
}

func pkr_run_cmd_update(cmd *commander.Command, args []string) error {
	var err error

	cfgtype := cmd.Flag.Lookup("type").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)

	switch len(args) {
	case 0:
		// no-op
	default:
		return fmt.Errorf("pkr: invalid number of arguments. expected none. got=%d (%v)",
			len(args),
			args,
		)
	}

	cfg := NewConfig(cfgtype)
	ctx, err := New(cfg, debug)
	if err != nil {
		return err
	}
	defer ctx.Close()

	ctx.msg.Infof("updating RPMs\n")
	checkOnly := false
	err = ctx.Update(checkOnly)
	return err
}