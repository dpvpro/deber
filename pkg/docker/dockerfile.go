package docker

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// DockerfileTemplate struct defines parameters passed to
// dockerfile template.
type DockerfileTemplate struct {
	From       string
	ArchiveDir string
	SourceDir  string
	Packages   string
	Backports  string
}

const dockerfileTemplate = `
# From which Docker image do we start?
FROM {{ .From }}

# Remove not needed apt configs.
RUN rm /etc/apt/apt.conf.d/*

# Run apt without confirmations.
RUN echo "APT::Get::Assume-Yes "true";" > /etc/apt/apt.conf.d/00noconfirm

# Set debconf to be non interactive
RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections

# Backports pinning
{{ .Backports }}

# Install required packages
RUN apt-get update && \
	apt-get install --no-install-recommends -y {{ .Packages }}

# Add local apt repository.
RUN mkdir -p {{ .ArchiveDir }} && \
    touch {{ .ArchiveDir }}/Packages && \
    echo "deb [trusted=yes] file://{{ .ArchiveDir }} ./" > /etc/apt/sources.list.d/a.list

# Set working directory.
WORKDIR {{ .SourceDir }}

# Sleep all the time and just wait for commands.
CMD ["sleep", "inf"]
`

func dockerfileParse(from string) (string, error) {
	t := DockerfileTemplate{
		From:       from,
		ArchiveDir: ContainerArchiveDir,
		SourceDir:  ContainerSourceDir,
		Packages:   "build-essential devscripts debhelper lintian fakeroot",
	}

	dist := strings.Split(from, ":")[1]

	if strings.Contains(dist, "-backports") {
		t.Backports = fmt.Sprintf(
			"RUN printf \"%s\" > %s",
			"Package: *\\nPin: release a="+dist+"\\nPin-Priority: 800",
			"/etc/apt/preferences.d/backports",
		)
	}

	temp, err := template.New("dockerfile").Parse(dockerfileTemplate)
	if err != nil {
		return "", err
	}

	buffer := new(bytes.Buffer)
	err = temp.Execute(buffer, t)
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}
