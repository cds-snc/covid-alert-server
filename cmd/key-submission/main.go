package main

import (
	"github.com/cds-snc/covid-alert-server/pkg/app"
	"github.com/cds-snc/covid-alert-server/pkg/cmd"
)

func main() {
	cmd.RunAndWait(
		app.NewBuilder().
			WithTestTools().
			WithSubmission())
}
