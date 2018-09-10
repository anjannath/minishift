// +build !systemtray

/*
Copyright (C) 2018 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package systemtray

import (
	"fmt"
	goos "os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/anjannath/systray"
	"github.com/docker/machine/libmachine"
	libmachineState "github.com/docker/machine/libmachine/state"
	"github.com/golang/glog"
	"github.com/minishift/minishift/cmd/minishift/state"
	"github.com/minishift/minishift/pkg/minikube/cluster"
	"github.com/minishift/minishift/pkg/minishift/profile"
	"github.com/minishift/minishift/pkg/minishift/systemtray/icon"
	"github.com/minishift/minishift/pkg/util/os"
	"github.com/minishift/minishift/pkg/util/slice"
)

const (
	START = "Start"
	STOP  = "Stop"
	EXIT  = "Exit"

	START_PROFILE int = iota
	STOP_PROFILE
	STOPPED
	RUNNING
	DOES_NOT_EXIST
)

var (
	submenus            = make(map[string]*systray.MenuItem)
	submenusToMenuItems = make(map[string]MenuAction)

	profiles        []string
	profileMenuList []*systray.MenuItem

	submenusLock            sync.RWMutex
	submenusToMenuItemsLock sync.RWMutex
)

type MenuAction struct {
	start *systray.MenuItem
	stop  *systray.MenuItem
}

func OnReady() {
	systray.SetIcon(icon.TrayIcon)
	exit := systray.AddMenuItem(EXIT, "", 0)
	systray.AddSeparator()
	profiles = profile.GetProfileList()
	for _, profile := range profiles {
		submenu := systray.AddSubMenu(profile)
		startMenu := submenu.AddSubMenuItem(START, "", 0)
		stopMenu := submenu.AddSubMenuItem(STOP, "", 0)
		submenus[profile] = submenu
		submenusToMenuItems[profile] = MenuAction{start: startMenu, stop: stopMenu}
	}

	go func() {
		<-exit.OnClickCh()
		systray.Quit()
	}()

	for k, v := range submenusToMenuItems {
		go startStopHandler(icon.Running, k, v.start, START_PROFILE)
		go startStopHandler(icon.Stopped, k, v.stop, STOP_PROFILE)
	}

	go addNewProfilesToTray()

	go removeDeletedProfilesFromTray()

	go updateProfileStatus()
}

func OnExit() {
	return
}

func getStatus(profileName string) int {
	api := libmachine.NewClient(state.InstanceDirs.Home, state.InstanceDirs.Certs)
	defer api.Close()

	vmStatus, err := cluster.GetHostStatus(api, profileName)
	fmt.Println("Getstatus:", vmStatus)
	if err != nil {
		return DOES_NOT_EXIST
	}

	if vmStatus == libmachineState.Running.String() {
		return RUNNING
	}
	if vmStatus == libmachineState.Stopped.String() {
		return STOPPED
	}
	return 0
}

// Add newly created profiles to the tray
func addNewProfilesToTray() {
	for {
		time.Sleep(40 * time.Second)

		newProfilesList := profile.GetProfileList()
		for _, profile := range newProfilesList {
			submenusLock.Lock()
			if _, ok := submenus[profile]; ok {
				submenusLock.Unlock()
				continue
			} else {
				submenu := systray.AddSubMenu(profile)
				submenus[profile] = submenu
				submenusLock.Unlock()
				startMenu := submenu.AddSubMenuItem(START, "", 0)
				stopMenu := submenu.AddSubMenuItem(STOP, "", 0)
				submenusToMenuItemsLock.Lock()
				ma := MenuAction{start: startMenu, stop: stopMenu}
				submenusToMenuItems[profile] = ma
				submenusToMenuItemsLock.Unlock()

				go startStopHandler(icon.Running, profile, ma.start, START_PROFILE)

				go startStopHandler(icon.Stopped, profile, ma.stop, STOP_PROFILE)
			}
		}
	}
}

// Remove deleted profiles from tray
func removeDeletedProfilesFromTray() {
	for {
		time.Sleep(30 * time.Second)
		newProfileList := profile.GetProfileList()
		for k := range submenus {
			submenusLock.Lock()
			if exists, _ := slice.ItemExists(newProfileList, k); exists {
				submenusLock.Unlock()
				continue
			} else {
				submenus[k].Hide()
				delete(submenus, k)
				submenusLock.Unlock()
				if _, ok := submenusToMenuItems[k]; ok {
					submenusToMenuItemsLock.Lock()
					delete(submenusToMenuItems, k)
					submenusToMenuItemsLock.Unlock()
				}
			}
		}
	}
}

// stopProfile stops a profile when clicked on the stop menuItem
func stopProfile(profileName string) error {
	minishiftBinary, _ := os.CurrentExecutable()

	if runtime.GOOS == "windows" {
		cmd, err := exec.LookPath("cmd.exe")
		if err != nil {
			if glog.V(3) {
				fmt.Println("Could not find cmd.exe in path")
				return fmt.Errorf("%v", err)
			}
		}
		args := []string{"/C", "start", minishiftBinary, "stop", "--profile", profileName}
		command := exec.Command(cmd, args...)
		return command.Run()
	}

	if runtime.GOOS == "darwin" {
		var stopCommandString = fmt.Sprintf(minishiftBinary + " stop --profile " + profileName)
		stopFilePath := filepath.Join(goos.TempDir(), "minishift.stop")
		fmt.Println(stopFilePath)
		fmt.Println(stopCommandString)

		f, err := goos.Create(stopFilePath)
		if err != nil {
			return err
		}
		if _, err = f.WriteString(stopCommandString); err != nil {
			return err
		}
		if err = f.Chmod(0744); err != nil {
			return err
		}
		args := []string{"-F", "-a", "Terminal.app", stopFilePath}
		cmd, err := exec.LookPath("open")
		if err != nil {
			if glog.V(3) {
				fmt.Println("Could not find open in path")
				return fmt.Errorf("%v", err)
			}
		}
		command := exec.Command(cmd, args...)
		return command.Run()
	}
	return nil
}

// startProfile starts a profile when clicked on the start menuItem
func startProfile(profileName string) error {
	minishiftBinary, _ := os.CurrentExecutable()
	fmt.Println(minishiftBinary)
	if runtime.GOOS == "windows" {
		cmd, err := exec.LookPath("cmd.exe")
		if err != nil {
			if glog.V(3) {
				fmt.Println("Could not find cmd.exe")
				return fmt.Errorf("%v", err)
			}
		}
		args := []string{"/C", "start", minishiftBinary, "start", "--profile", profileName}
		command := exec.Command(cmd, args...)
		return command.Run()
	}
	if runtime.GOOS == "darwin" {
		var stopCommandString = fmt.Sprintf(minishiftBinary + " start --profile " + profileName)
		stopFilePath := filepath.Join(goos.TempDir(), "minishift.start")
		fmt.Println(stopFilePath)
		fmt.Println(stopCommandString)

		f, err := goos.Create(stopFilePath)
		if err != nil {
			return err
		}
		if _, err = f.WriteString(stopCommandString); err != nil {
			return err
		}
		if err = f.Chmod(0744); err != nil {
			return err
		}

		args := []string{"-F", "-a", "Terminal.app", stopFilePath}
		cmd, err := exec.LookPath("open")
		if err != nil {
			if glog.V(3) {
				fmt.Println("Could not find open in path")
				return fmt.Errorf("%v", err)
			}
		}
		command := exec.Command(cmd, args...)
		return command.Run()
	}
	return nil
}

// updateProfileStatus updates the menu bitmap to reflact the state of
// machine, green: running, red: stoppped, grey: does not exist
func updateProfileStatus() {
	for {
		time.Sleep(20 * time.Second)
		submenusLock.Lock()
		for k, v := range submenus {
			status := getStatus(k)
			fmt.Println("updateProfile:", status, k)
			if status == DOES_NOT_EXIST {
				v.AddBitmap(icon.DoesNotExist)
			}
			if status == RUNNING {
				v.AddBitmap(icon.Running)
			}
			if status == STOPPED {
				v.AddBitmap(icon.Stopped)
			}
		}
		submenusLock.Unlock()
	}
}

func startStopHandler(iconData []byte, submenu string, m *systray.MenuItem, action int) {
	var err error
	for {
		<-m.OnClickCh()
		if action == START_PROFILE {
			err = startProfile(submenu)
		} else {
			err = stopProfile(submenu)
		}
		if err == nil {
			submenusLock.Lock()
			submenus[submenu].AddBitmap(iconData)
			submenusLock.Unlock()
		}
	}
}
