package app

import (
	"errors"
	"fmt"
	doc "github.com/dawidd6/deber/pkg/docker"
	"github.com/dawidd6/deber/pkg/naming"
	"github.com/spf13/cobra"
	"os"
	"pault.ag/go/debian/changelog"
	"strings"
)

func run(cmd *cobra.Command, args []string) error {
	steps := map[string]func(*doc.Docker, *naming.Naming) error{
		"build":   runBuild,
		"create":  runCreate,
		"start":   runStart,
		"tarball": runTarball,
		"scan":    runScan,
		"update":  runUpdate,
		"deps":    runDeps,
		"package": runPackage,
		"test":    runTest,
		"stop":    runStop,
		"remove":  runRemove,
		"archive": runArchive,
	}
	keys := []string{
		"build",
		"create",
		"start",
		"tarball",
		"scan",
		"update",
		"deps",
		"package",
		"test",
		"stop",
		"remove",
		"archive",
	}

	log.Info("Parsing Debian changelog")
	debian, err := changelog.ParseFileOne("debian/changelog")
	if err != nil {
		return err
	}

	log.Info("Connecting with Docker")
	docker, err := doc.New()
	if err != nil {
		return err
	}

	tarball, err := getTarball(debian.Source, debian.Version.Version)
	if err != nil && !debian.Version.IsNative() {
		return err
	}

	name := naming.New(
		cmd.Use,
		debian.Target,
		debian.Source,
		debian.Version.String(),
		tarball,
	)

	if include != "" && exclude != "" {
		return errors.New("can't specify --include and --exclude together")
	}

	if include != "" {
		for key := range steps {
			if !strings.Contains(include, key) {
				delete(steps, key)
			}
		}
	}

	if exclude != "" {
		for key := range steps {
			if strings.Contains(exclude, key) {
				delete(steps, key)
			}
		}
	}

	for i := range keys {
		f, ok := steps[keys[i]]
		if ok {
			err := f(docker, name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func runBuild(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Building image")

	isImageBuilt, err := docker.IsImageBuilt(name.Image())
	if err != nil {
		return err
	}
	if isImageBuilt {
		return nil
	}

	// TODO strip -security suffix, because there are no images available like this
	from := ""

	for _, o := range []string{"debian", "ubuntu"} {
		tags, err := doc.GetTags(o)
		if err != nil {
			return err
		}

		for _, tag := range tags {
			if tag.Name == name.Dist() {
				from = fmt.Sprintf("%s:%s", o, name.Dist())
				break
			}
		}

		if from != "" {
			break
		}
	}

	if from == "" {
		return errors.New("dist image not found")
	}

	err = docker.BuildImage(name.Image(), from)
	if err != nil {
		return err
	}

	return nil
}

func runCreate(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Creating container")

	isContainerCreated, err := docker.IsContainerCreated(name.Container())
	if err != nil {
		return err
	}
	if isContainerCreated {
		return nil
	}

	err = docker.CreateContainer(name)
	if err != nil {
		return err
	}

	return nil
}

func runStart(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Starting container")

	isContainerStarted, err := docker.IsContainerStarted(name.Container())
	if err != nil {
		return err
	}
	if isContainerStarted {
		return nil
	}

	err = docker.StartContainer(name.Container())
	if err != nil {
		return err
	}

	return nil
}

func runTarball(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Moving tarball")

	if name.Tarball() != "" {
		err := os.Rename(name.HostSourceSourceTarballFile(), name.HostBuildTargetTarballFile())
		if err != nil {
			return err
		}
	}

	return nil
}

func runScan(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Scanning archive")

	err := docker.ExecContainer(name.Container(), "scan")
	if err != nil {
		return err
	}

	return nil
}

func runUpdate(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Updating cache")

	err := docker.ExecContainer(name.Container(), "sudo", "apt-get", "update")
	if err != nil {
		return err
	}

	return nil
}

func runDeps(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Installing dependencies")

	err := docker.ExecContainer(name.Container(), "sudo", "mk-build-deps", "-ri", "-t", "apty")
	if err != nil {
		return err
	}

	return nil
}

func runPackage(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Packaging software")

	file := fmt.Sprintf("%s/%s", name.HostArchiveFromDir(), "Packages")
	info, err := os.Stat(file)
	if info == nil {
		_, err := os.Create(file)
		if err != nil {
			return err
		}
	}

	flags := strings.Split(dpkgFlags, " ")
	command := append([]string{"dpkg-buildpackage"}, flags...)
	err = docker.ExecContainer(name.Container(), command...)
	if err != nil {
		return err
	}

	return nil
}

func runTest(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Testing package")

	err := docker.ExecContainer(name.Container(), "debc")
	if err != nil {
		return err
	}

	err = docker.ExecContainer(name.Container(), "sudo", "debi", "--with-depends", "--tool", "apty")
	if err != nil {
		return err
	}

	flags := strings.Split(lintianFlags, " ")
	command := append([]string{"lintian"}, flags...)
	err = docker.ExecContainer(name.Container(), command...)
	if err != nil {
		return err
	}

	return nil
}

func runStop(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Stopping container")

	isContainerStopped, err := docker.IsContainerStopped(name.Container())
	if err != nil {

		return err
	}
	if isContainerStopped {
		return nil
	}

	err = docker.StopContainer(name.Container())
	if err != nil {
		return err
	}

	return nil
}

func runRemove(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Removing container")

	isContainerCreated, err := docker.IsContainerCreated(name.Container())
	if err != nil {
		return err
	}
	if !isContainerCreated {
		return nil
	}

	err = docker.RemoveContainer(name.Container())
	if err != nil {
		return err
	}

	return nil
}

func runArchive(docker *doc.Docker, name *naming.Naming) error {
	log.Info("Archiving build")

	info, err := os.Stat(name.HostArchiveFromOutputDir())
	if info != nil {
		return nil
	}

	err = os.Rename(name.HostBuildOutputDir(), name.HostArchiveFromOutputDir())
	if err != nil {
		return err
	}

	return nil
}
