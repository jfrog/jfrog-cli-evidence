package reports

import "github.com/jfrog/jfrog-cli-evidence/evidence/model"

type ReportPrinter interface {
	Print(result *model.VerificationResponse) error
}
