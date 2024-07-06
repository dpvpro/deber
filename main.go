package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dpvpro/deber/pkg/docker"
	"github.com/dpvpro/deber/pkg/log"
	"github.com/dpvpro/deber/pkg/naming"
	"github.com/dpvpro/deber/pkg/steps"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"pault.ag/go/debian/changelog"
)

const (
	// Program is the name of program
	Program = "deber"
	// Version of program
	Version = "1.2.0"
	// Description of program
	Description = "Debian packaging with Docker."
)

var (
	buildDir     = pflag.StringP("build-dir", "B", "", "where to place build stuff")
	cacheDir     = pflag.StringP("cache-dir", "C", "", "where to place cached stuff")
	distribution = pflag.StringP("distribution", "d", "", "override target distribution")
	packages     = pflag.StringArrayP("package", "p", nil, "additional packages to be installed in container (either single .deb or a directory)")
	age          = pflag.DurationP("age", "a", time.Hour*24*14, "time after which image will be refreshed")
	network      = pflag.BoolP("network", "n", false, "allow network access during package build")
	shell        = pflag.BoolP("shell", "s", false, "launch interactive shell in container")
	dpkgFlags    = pflag.StringP("dpkg-flags", "D", "-tc", "additional flags to be passed to dpkg-buildpackage in container")
	lintianFlags = pflag.StringP("lintian-flags", "L", "-i -I", "additional flags to be passed to lintian in container")
	noLintian    = pflag.BoolP("no-lintian", "l", false, "don't run lintian in container")
	noLogColor   = pflag.BoolP("no-log-color", "c", false, "do not colorize log output")
	noRemove     = pflag.BoolP("no-remove", "r", false, "do not remove container at the end of the process")
)

func main() {
	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s [FLAGS ...]", Program),
		Short:   Description,
		Version: Version,
		RunE:    run,
	}

	cmd.SetHelpCommand(&cobra.Command{Hidden: true})
	cmd.DisableFlagsInUseLine = true
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	log.NoColor = *noLogColor

	dock, err := docker.New()
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// home, err := os.UserHomeDir()
	// if err != nil {
	// 	return err
	// }

	home := filepath.Join(os.TempDir(), Program)
	err = os.MkdirAll(home, os.ModePerm)
	if err != nil {
		return err
	}

	source := filepath.Join(home, "sources")
	err = os.MkdirAll(source, os.ModePerm)
	if err != nil {
		return err
	}


	if *buildDir == "" {
		*buildDir = filepath.Join(home, "builddir")
		err = os.MkdirAll(*buildDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	if *cacheDir == "" {
  	*cacheDir = filepath.Join(home, "cachedir")
  	err = os.MkdirAll(*cacheDir, os.ModePerm)
  	if err != nil {
  		return err
  	}
	}

	path := filepath.Join(cwd, "debian/changelog")
	ch, err := changelog.ParseFileOne(path)
	if err != nil {
		return err
	}

	if *distribution == "" {
		*distribution = ch.Target
	}

	namingArgs := naming.Args{
		Prefix:         Program,
		Source:         ch.Source,
		Version:        ch.Version.String(),
		Upstream:       ch.Version.Version,
		Target:         *distribution,
		SourceBaseDir:  cwd,
		BuildBaseDir:   *buildDir,
		CacheBaseDir:   *cacheDir,
		PackagesBaseDir: filepath.Join(home, "packages"),
	}
	n := naming.New(namingArgs)

	err = steps.Build(dock, n, *age)
	if err != nil {
		return err
	}

	err = steps.Create(dock, n, *packages)
	if err != nil {
		return err
	}

	err = steps.Start(dock, n)
	if err != nil {
		return err
	}

	if *shell {
		return steps.ShellOptional(dock, n)
	}

	err = steps.Tarball(n)
	if err != nil {
		return err
	}

	err = steps.Depends(dock, n, *packages)
	if err != nil {
		return err
	}

	err = steps.Package(dock, n, *dpkgFlags, *network)
	if err != nil {
		return err
	}

	err = steps.Test(dock, n, *lintianFlags, *noLintian)
	if err != nil {
		return err
	}

	err = steps.Archive(n)
	if err != nil {
		return err
	}

	err = steps.Stop(dock, n)
	if err != nil {
		return err
	}

	if *noRemove {
		return nil
	}
	return steps.Remove(dock, n)
}
