package installer

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

type diskInstallChoices struct {
	disk         Disk
	hostname     string
	passwordHash string
	subnet       string
}

func checkNav(value string) error {
	switch strings.ToLower(value) {
	case "back":
		return errBack
	case "restart":
		return errRestart
	case "cancel":
		return errCancel
	default:
		return nil
	}
}

func askNav(c console, prompt string) (string, error) {
	val, err := c.ask(prompt)
	if err != nil {
		return "", err
	}
	if navErr := checkNav(val); navErr != nil {
		return "", navErr
	}
	return val, nil
}

func askSecretNav(c console, prompt string) (string, error) {
	val, err := c.secret(prompt)
	if err != nil {
		return "", err
	}
	if navErr := checkNav(val); navErr != nil {
		return "", navErr
	}
	return val, nil
}

func stepNetwork(ctx context.Context, c console, run commandRunner) error {
	err := c.networkWith(ctx, run)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, errBack), errors.Is(err, errCancel):
		return errCancel
	default:
		return err
	}
}

func handleDiskInspectFailure(c console) error {
	c.page("Step 2 of 5 — Installation disk")
	c.print("Could not inspect disks. No disk installation started.")
	_, err := askNav(c, "Type retry, back, restart, or cancel")
	return err
}

func printDiskList(c console, disks []Disk, feedback string) {
	c.page("Step 2 of 5 — Installation disk")
	if feedback != "" {
		c.print("%s", feedback)
		c.print("")
	}
	for i, disk := range disks {
		c.print("%d. %s", i+1, diskSummary(disk.Device))
		for _, child := range disk.Device.Children {
			c.print("   %q: %.1f GiB, filesystem %q", child.Name, float64(child.Size)/(1<<30), child.FSType)
		}
		if disk.Blocked != "" {
			c.print("   Unavailable: %s", disk.Blocked)
		}
	}
}

func selectDisk(disks []Disk, selected string) (Disk, string, bool) {
	index, err := strconv.Atoi(selected)
	if err != nil || index < 1 || index > len(disks) {
		return Disk{}, "Choose an available disk number.", false
	}
	disk := disks[index-1]
	return disk, "", true
}

func stepDisk(ctx context.Context, c console, run commandRunner, inspect func(context.Context, commandRunner) ([]Disk, error)) (Disk, error) {
	feedback := ""
	for {
		disks, err := inspect(ctx, run)
		if err != nil {
			if navErr := handleDiskInspectFailure(c); navErr != nil {
				return Disk{}, navErr
			}
			continue
		}
		printDiskList(c, disks, feedback)
		feedback = ""
		selected, err := askNav(c, "Disk number, back, restart, or cancel")
		if err != nil {
			return Disk{}, err
		}
		disk, errFeedback, ok := selectDisk(disks, selected)
		if !ok {
			feedback = errFeedback
			continue
		}
		return disk, nil
	}
}

func stepHostname(c console, selectedDisk Disk, currentHostname string) (string, error) {
	feedback := ""
	for {
		c.page("Step 3 of 5 — Hostname")
		if feedback != "" {
			c.print("%s", feedback)
			c.print("")
			feedback = ""
		}
		c.print("Selected disk: %s", diskSummary(selectedDisk.Device))
		hostnameDefault := currentHostname
		if hostnameDefault == "" {
			hostnameDefault = "soda"
		}
		value, err := askNav(c, "Hostname ["+hostnameDefault+"], back, restart, or cancel")
		if err != nil {
			return "", err
		}
		if value == "" {
			value = hostnameDefault
		}
		if !Hostname(value) {
			feedback = "Use lowercase letters, digits, dots, and interior hyphens."
			continue
		}
		return value, nil
	}
}

func validPassword(password, confirmation string) bool {
	return utf8.ValidString(password) && password != "" && password == confirmation
}

func hashPassword(ctx context.Context, run commandRunner, password string) (string, error) {
	hash, err := run(ctx, "openssl", []string{"passwd", "-6", "-stdin"}, strings.NewReader(password+"\n"))
	if err != nil {
		return "", errors.New("password hashing failed")
	}
	return strings.TrimSpace(string(hash)), nil
}

func stepPassword(ctx context.Context, c console, run commandRunner) (string, error) {
	feedback := ""
	for {
		c.page("Step 4 of 5 — Native operator password")
		if feedback != "" {
			c.print("%s", feedback)
			c.print("")
			feedback = ""
		}
		c.print("This password is for root login after reboot, local console and SSH.")
		c.print("Enroll a key later and disable password logins yourself to go key-only.")
		c.print("Type back, restart, or cancel in a password field to navigate.")
		password, err := askSecretNav(c, "Password")
		if err != nil {
			return "", err
		}
		confirmation, err := askSecretNav(c, "Confirm password")
		if err != nil {
			return "", err
		}
		if !validPassword(password, confirmation) {
			feedback = "Passwords must match and must not be empty."
			continue
		}
		return hashPassword(ctx, run, password)
	}
}

func stepSubnet(c console, currentSubnet string) (string, error) {
	feedback := ""
	for {
		c.page("Step 5 of 5 — Project network and review")
		if feedback != "" {
			c.print("%s", feedback)
			c.print("")
			feedback = ""
		}
		c.print("Developer client routing is configured separately. Any canonical IPv4 range is accepted, including public or overlapping ranges.")
		subnetDefault := currentSubnet
		if subnetDefault == "" {
			subnetDefault = "10.89.0.0/24"
		}
		subnet, err := askNav(c, "Project IPv4 subnet ["+subnetDefault+"], back, restart, or cancel")
		if err != nil {
			return "", err
		}
		if subnet == "" {
			subnet = subnetDefault
		}
		if err := ProjectSubnet(subnet); err != nil {
			feedback = "Invalid subnet: " + err.Error()
			continue
		}
		return subnet, nil
	}
}

func printFinalReview(c console, choices diskInstallChoices, payloadBytes uint64) {
	c.page("Final review")
	c.print("ERASE ALL DATA on:")
	c.print("  %s", diskSummary(choices.disk.Device))
	if choices.disk.Blocked != "" {
		c.print("  Installer note: %s. Typing ERASE still wipes it.", choices.disk.Blocked)
	}
	c.print("Hostname: %s", choices.hostname)
	c.print("Project subnet: %s", choices.subnet)
	c.print("Operator access: root password (local console and SSH)")
	c.print("Included Soda payload: %.1f MiB verified", float64(payloadBytes)/(1<<20))
	c.print("Network settings will be copied to the installed system.")
	c.print("After writing, follow the completion screen for media removal and next steps.")
}

func confirmFinalReview(c console, diskName string) error {
	phrase := "ERASE " + diskName
	for {
		answer, err := askNav(c, "Type exactly "+phrase+", back, restart, or cancel")
		if err != nil {
			return err
		}
		if answer == phrase {
			return nil
		}
		c.print("Confirmation did not match. No disk writing started.")
	}
}

func stepSubnetAndReview(ctx context.Context, c console, run commandRunner, choices *diskInstallChoices, payloadBytes uint64) error {
	for {
		subnet, err := stepSubnet(c, choices.subnet)
		if err != nil {
			return err
		}
		choices.subnet = subnet
		printFinalReview(c, *choices, payloadBytes)
		err = confirmFinalReview(c, choices.disk.Device.Name)
		if errors.Is(err, errBack) {
			continue
		}
		return err
	}
}

func dispatchInstallStep(ctx context.Context, c console, run commandRunner, inspect func(context.Context, commandRunner) ([]Disk, error), payloadBytes uint64, step int, result *diskInstallChoices) error {
	switch step {
	case 0:
		return stepNetwork(ctx, c, run)
	case 1:
		var err error
		result.disk, err = stepDisk(ctx, c, run, inspect)
		return err
	case 2:
		var err error
		result.hostname, err = stepHostname(c, result.disk, result.hostname)
		return err
	case 3:
		var err error
		result.passwordHash, err = stepPassword(ctx, c, run)
		return err
	case 4:
		err := stepSubnetAndReview(ctx, c, run, result, payloadBytes)
		if err != nil {
			result.passwordHash = ""
		}
		return err
	default:
		return nil
	}
}

func collectDiskInstallChoices(ctx context.Context, c console, run commandRunner, inspect func(context.Context, commandRunner) ([]Disk, error), payloadBytes uint64) (diskInstallChoices, error) {
	var result diskInstallChoices
	step := 0
	for {
		err := dispatchInstallStep(ctx, c, run, inspect, payloadBytes, step, &result)
		if errors.Is(err, errBack) {
			step--
		} else if err != nil {
			return result, err
		} else if step == 4 {
			return result, nil
		} else {
			step++
		}
	}
}
