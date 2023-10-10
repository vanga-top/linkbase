package cli

import (
	"fmt"
	"github.com/linkbase/middleware/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"os"
	"testing"
)

func TestAppRun(t *testing.T) {
	app := &cli.App{
		Name:  "greet",
		Usage: "fight the loneliness!",
		Action: func(*cli.Context) error {
			fmt.Println("Hello friend!")
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Error("", zap.Error(err))
	}
}
