package configdiff

// Commandes standard (équipements Cisco-like).
const (
	cmdRunning = "show running-config"
	cmdStartup = "show startup-config"
)

// Fetch récupère la running-config et la startup-config via le runner.
func Fetch(r Runner) (running, startup string, err error) {
	running, err = r.Run(cmdRunning)
	if err != nil {
		return "", "", err
	}
	startup, err = r.Run(cmdStartup)
	if err != nil {
		return "", "", err
	}
	return running, startup, nil
}
