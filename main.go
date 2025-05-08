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
	Version = "1.4.9"
	// Description of program
	Description = "Debian packaging with Docker"
)

var (
	buildDir     = pflag.StringP("build-dir", "B", "", "where to place build stuff")
	cacheDir     = pflag.StringP("cache-dir", "C", "", "where to place cached stuff")
	systemDir    = pflag.StringP("system-dir", "S", "", "system directory for deber")
	distribution = pflag.StringP("distribution", "T", "", "override target distribution")
	dpkgFlags    = pflag.StringP("dpkg-flags", "D", "-b -uc -tc", "additional flags to be passed to dpkg-buildpackage in container")
	lintianFlags = pflag.StringP("lintian-flags", "L", "-i -I", "additional flags to be passed to lintian in container")
	packages     = pflag.StringArrayP("package", "P", nil, "additional packages to be installed in container (either single .deb or a directory)")
	age          = pflag.DurationP("age", "a", time.Hour*24*14, "time after which image will be refreshed")
	network      = pflag.BoolP("network", "n", false, "allow network access during package build")
	shell        = pflag.BoolP("shell", "s", false, "launch interactive shell in container")
	lintian      = pflag.BoolP("lintian", "l", false, "run lintian in container")
	noTest       = pflag.BoolP("no-test", "t", true, "do not test when building package")
	noLogColor   = pflag.BoolP("no-log-color", "", false, "do not colorize log output")
	noRemove     = pflag.BoolP("no-remove", "", false, "do not remove container at the end of the process")

	packagesDir string
)

func main() {
	
	cmd := &cobra.Command{
		Use:                   fmt.Sprintf("%s [FLAGS ...]", Program),
		Short:                 Description,
		Version:               Version,
		RunE:                  run,
		SilenceUsage:          true,
		SilenceErrors:         true,
		Hidden:                true,
		DisableFlagsInUseLine: true,
	}

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

	if *systemDir == "" {
		*systemDir = filepath.Join(os.TempDir(), Program)
	}
	
	packagesDir = filepath.Join(*systemDir, "packages")
	sources := filepath.Join(*systemDir, "sources")

	if *buildDir == "" {
		*buildDir = filepath.Join(*systemDir, "builddir")
	}

	if *cacheDir == "" {
		*cacheDir = filepath.Join(*systemDir, "cachedir")
	}

	err = os.MkdirAll(*systemDir, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(packagesDir, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(sources, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(*buildDir, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(*cacheDir, os.ModePerm)
	if err != nil {
		return err
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
		Prefix:          Program,
		Source:          ch.Source,
		Version:         ch.Version.String(),
		Upstream:        ch.Version.Version,
		Target:          *distribution,
		SourceBaseDir:   cwd,
		BuildBaseDir:    *buildDir,
		CacheBaseDir:    *cacheDir,
		PackagesBaseDir: packagesDir,
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

	err = steps.Package(dock, n, *dpkgFlags, *network, *noTest)
	if err != nil {
		errRemove := steps.Remove(dock, n)
		if errRemove != nil {
			 fmt.Printf("%s", errRemove)
		}
		return err
	}

	err = steps.Test(dock, n, *lintianFlags, *lintian)
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
