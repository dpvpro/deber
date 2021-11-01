// Package dockerfile includes template Dockerfile
package dockerfile

import (
	"bytes"
	"github.com/dawidd6/deber/pkg/naming"
	"text/template"
)

// Template struct defines parameters passed to
// dockerfile template.
type Template struct {
	// Repo is the image repository
	Repo string
	// Tag is the image tag
	Tag string
	// SourceDir = /build/source
	SourceDir string
}

const dockerfileTemplate = `
# From which Docker image do we start?
FROM {{ .Repo }}:{{ .Tag }}

# Remove not needed apt configs.
RUN rm /etc/apt/apt.conf.d/*

# Run apt without confirmations.
RUN echo "APT::Get::Assume-Yes "true";" > /etc/apt/apt.conf.d/00noconfirm

# Set debconf to be non interactive.
RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections

# Install required packages.
RUN rm -f /etc/apt/sources.list && \
	echo 'deb http://192.168.11.118/ bullseye main' | tee /etc/apt/sources.list && \
	echo 'deb-src http://192.168.11.118/ bullseye main' | tee -a /etc/apt/sources.list && \
	apt-get update --allow-insecure-repositories && \
	apt-get install wget gnupg2 tar bzip2 mc htop -y --allow-unauthenticated && \
	wget -qO - http://192.168.11.118/veil-repo-key.gpg | apt-key add -


# upgrade/downgrade пакетов до версий в зеркале
RUN printf "Package: *\nPin: release a=bullseye\nPin-Priority: 1001\n" > /etc/apt/preferences

RUN apt-get update && \
	apt-get dist-upgrade -y --allow-downgrades && \
	rm -f /etc/apt/preferences && \
	apt-get update

# Pin local repo (apt-get -t option pins with priority 990 too).
RUN printf "Package: *\nPin: origin \"\"\nPin-Priority: 990\n" > /etc/apt/preferences.d/00a

RUN cd /root && wget -q http://192.168.10.144/files/svace/svace-3.1.1-x64-linux.tar.bz2 && \
	tar xvf svace-3.1.1-x64-linux.tar.bz2 && \
	mv svace-3.1.1-x64-linux /opt/svace-311

ENV PATH "$PATH:/opt/svace-311/bin/"

RUN apt-get update && \
	apt-get install --no-install-recommends -y \
	build-essential devscripts debhelper lintian fakeroot dpkg-dev libperl-dev libssl-dev bison wget

RUN	wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - && \
	echo "deb http://apt.postgresql.org/pub/repos/apt bullseye-pgdg main" > /etc/apt/sources.list.d/pgdg.list && \
	echo "deb-src http://apt.postgresql.org/pub/repos/apt bullseye-pgdg main" >> /etc/apt/sources.list.d/pgdg.list

RUN apt update && apt install postgresql-all

# Set working directory.
WORKDIR {{ .SourceDir }}

# Sleep all the time and just wait for commands.
CMD ["sleep", "4h"]
`

// Parse function returns ready to use template
func Parse(repo, tag string) ([]byte, error) {
	t := Template{
		Repo:      repo,
		Tag:       tag,
		SourceDir: naming.ContainerSourceDir,
	}

	temp, err := template.New("dockerfile").Parse(dockerfileTemplate)
	if err != nil {
		return nil, err
	}

	buffer := new(bytes.Buffer)
	err = temp.Execute(buffer, t)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
