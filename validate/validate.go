package validate

import (
	"fmt"

	"github.com/urfave/cli"
)

func ValidateServerArgs(c *cli.Context) error {
	for _, p := range []string{"dictycontent-pass", "dictycontent-db", "dictycontent-user"} {
		if len(c.String(p)) == 0 {
			return cli.NewExitError(
				fmt.Sprintf("argument %s is missing", p),
				2,
			)
		}
	}
	return nil
}
