package githubrepository

import (
	"fmt"
	"time"
)

// Config is the configuration required to start the GitHub Controller.
type Config struct {
	Domain               string
	InsecureSSL          bool
	CallbackPort         string
	ReconciliationPeriod time.Duration
}

// PayloadURL is returns the fully qualified URL used to do payload callbacks to.
func (c Config) PayloadURL(owner, repo string) string {
	scheme := "https://"
	if c.InsecureSSL {
		scheme = "http://"
	}

	return fmt.Sprintf("%s%s/payload/%s/%s", scheme, c.Domain, owner, repo)
}
