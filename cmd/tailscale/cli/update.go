// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"runtime"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"tailscale.com/clientupdate"
	"tailscale.com/version"
	"tailscale.com/version/distro"
)

var updateCmd = &ffcli.Command{
	Name:       "update",
	ShortUsage: "update",
	ShortHelp:  "[ALPHA] Update Tailscale to the latest/different version",
	Exec:       runUpdate,
	FlagSet: (func() *flag.FlagSet {
		fs := newFlagSet("update")
		fs.BoolVar(&updateArgs.yes, "yes", false, "update without interactive prompts")
		fs.BoolVar(&updateArgs.dryRun, "dry-run", false, "print what update would do without doing it, or prompts")
		fs.BoolVar(&updateArgs.appStore, "app-store", false, "HIDDEN: check the App Store for updates, even if this is not an App Store install (for testing only)")
		// These flags are not supported on several systems that only provide
		// the latest version of Tailscale:
		//
		//  - Arch (and other pacman-based distros)
		//  - Alpine (and other apk-based distros)
		//  - FreeBSD (and other pkg-based distros)
		if distro.Get() != distro.Arch && distro.Get() != distro.Alpine && runtime.GOOS != "freebsd" {
			fs.StringVar(&updateArgs.track, "track", "", `which track to check for updates: "stable" or "unstable" (dev); empty means same as current`)
			fs.StringVar(&updateArgs.version, "version", "", `explicit version to update/downgrade to`)
		}
		return fs
	})(),
}

var updateArgs struct {
	yes      bool
	dryRun   bool
	appStore bool
	track    string // explicit track; empty means same as current
	version  string // explicit version; empty means auto
}

func runUpdate(ctx context.Context, args []string) error {
	if len(args) > 0 {
		return flag.ErrHelp
	}
	if updateArgs.version != "" && updateArgs.track != "" {
		return errors.New("cannot specify both --version and --track")
	}
	ver := updateArgs.version
	if updateArgs.track != "" {
		ver = updateArgs.track
	}
	err := clientupdate.Update(clientupdate.UpdateArgs{
		Version:  ver,
		AppStore: updateArgs.appStore,
		Logf:     func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
		Confirm:  confirmUpdate,
	})
	if errors.Is(err, errors.ErrUnsupported) {
		return errors.New("The 'update' command is not supported on this platform; see https://tailscale.com/s/client-updates")
	}
	return err
}

func confirmUpdate(ver string) bool {
	if updateArgs.yes {
		fmt.Printf("Updating Tailscale from %v to %v; --yes given, continuing without prompts.\n", version.Short(), ver)
		return true
	}

	if updateArgs.dryRun {
		fmt.Printf("Current: %v, Latest: %v\n", version.Short(), ver)
		return false
	}

	fmt.Printf("This will update Tailscale from %v to %v. Continue? [y/n] ", version.Short(), ver)
	var resp string
	fmt.Scanln(&resp)
	resp = strings.ToLower(resp)
	switch resp {
	case "y", "yes", "sure":
		return true
	}
	return false
}
