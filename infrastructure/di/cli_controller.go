package di

import (
	"github.com/ca-srg/tosage/interface/cli"
	"github.com/ca-srg/tosage/interface/presenter"
	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// newCLIController creates a new CLI controller
func newCLIController(
	ccService usecase.CcService,
	consolePresenter presenter.ConsolePresenter,
	jsonPresenter presenter.JSONPresenter,
) *cli.CLIController {
	return cli.NewCLIController(
		ccService,
		consolePresenter,
		jsonPresenter,
	)
}
