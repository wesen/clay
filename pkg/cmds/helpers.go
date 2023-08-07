package cmds

import (
	"context"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
)

func ListCommandsIntoProcessor(ctx context.Context, commands []cmds.Command, gp middlewares.Processor) error {
	for _, cmd := range commands {
		description := cmd.Description()
		for _, flag := range description.Flags {
			row := types.NewRow(types.MRP("command", description.Name), types.MRP("type", "flag"))
			types.SetFromStruct(row, flag, true)
			err := gp.AddRow(ctx, row)
			if err != nil {
				return err
			}
		}
		for _, arg := range description.Arguments {
			row := types.NewRow(types.MRP("command", description.Name), types.MRP("type", "argument"))
			types.SetFromStruct(row, arg, true)
			err := gp.AddRow(ctx, row)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
