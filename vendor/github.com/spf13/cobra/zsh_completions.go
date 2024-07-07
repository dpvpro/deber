// Copyright 2013-2023 The Cobra Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cobra

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/pflag"
)

const (
	zshCompArgumentAnnotation   = "cobra_annotations_zsh_completion_argument_annotation"
	zshCompArgumentFilenameComp = "cobra_annotations_zsh_completion_argument_file_completion"
	zshCompArgumentWordComp     = "cobra_annotations_zsh_completion_argument_word_completion"
	zshCompDirname              = "cobra_annotations_zsh_dirname"
)

var (
	zshCompFuncMap = template.FuncMap{
		"genZshFuncName":              zshCompGenFuncName,
		"extractFlags":                zshCompExtractFlag,
		"genFlagEntryForZshArguments": zshCompGenFlagEntryForArguments,
		"extractArgsCompletions":      zshCompExtractArgumentCompletionHintsForRendering,
	}
	zshCompletionText = `
{{/* should accept Command (that contains subcommands) as parameter */}}
{{define "argumentsC" -}}
{{ $cmdPath := genZshFuncName .}}
function {{$cmdPath}} {
  local -a commands

  _arguments -C \{{- range extractFlags .}}
    {{genFlagEntryForZshArguments .}} \{{- end}}
    "1: :->cmnds" \
    "*::arg:->args"

  case $state in
  cmnds)
    commands=({{range .Commands}}{{if not .Hidden}}
      "{{.Name}}:{{.Short}}"{{end}}{{end}}
    )
    _describe "command" commands
    ;;
  esac

  case "$words[1]" in {{- range .Commands}}{{if not .Hidden}}
  {{.Name}})
    {{$cmdPath}}_{{.Name}}
    ;;{{end}}{{end}}
  esac
}
{{range .Commands}}{{if not .Hidden}}
{{template "selectCmdTemplate" .}}
{{- end}}{{end}}
{{- end}}

{{/* should accept Command without subcommands as parameter */}}
{{define "arguments" -}}
function {{genZshFuncName .}} {
{{"  _arguments"}}{{range extractFlags .}} \
    {{genFlagEntryForZshArguments . -}}
{{end}}{{range extractArgsCompletions .}} \
    {{.}}{{end}}
}
{{end}}

{{/* dispatcher for commands with or without subcommands */}}
{{define "selectCmdTemplate" -}}
{{if .Hidden}}{{/* ignore hidden*/}}{{else -}}
{{if .Commands}}{{template "argumentsC" .}}{{else}}{{template "arguments" .}}{{end}}
{{- end}}
{{- end}}

{{/* template entry point */}}
{{define "Main" -}}
#compdef _{{.Name}} {{.Name}}

{{template "selectCmdTemplate" .}}
{{end}}
`
)

// zshCompArgsAnnotation is used to encode/decode zsh completion for
// arguments to/from Command.Annotations.
type zshCompArgsAnnotation map[int]zshCompArgHint

type zshCompArgHint struct {
	// Indicates the type of the completion to use. One of:
	// zshCompArgumentFilenameComp or zshCompArgumentWordComp
	Tipe string `json:"type"`

	// A value for the type above (globs for file completion or words)
	Options []string `json:"options"`
}

// GenZshCompletionFile generates zsh completion file.
func (c *Command) GenZshCompletionFile(filename string) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return c.GenZshCompletion(outFile)
}

func (c *Command) genZshCompletion(w io.Writer, includeDesc bool) error {
	buf := new(bytes.Buffer)
	genZshComp(buf, c.Name(), includeDesc)
	_, err := buf.WriteTo(w)
	return err
}

func genZshComp(buf io.StringWriter, name string, includeDesc bool) {
	compCmd := ShellCompRequestCmd
	if !includeDesc {
		compCmd = ShellCompNoDescRequestCmd
	}
	WriteStringAndCheck(buf, fmt.Sprintf(`#compdef %[1]s
compdef _%[1]s %[1]s

# zsh completion for %-36[1]s -*- shell-script -*-

__%[1]s_debug()
{
    local file="$BASH_COMP_DEBUG_FILE"
    if [[ -n ${file} ]]; then
        echo "$*" >> "${file}"
    fi
}

_%[1]s()
{
    local shellCompDirectiveError=%[3]d
    local shellCompDirectiveNoSpace=%[4]d
    local shellCompDirectiveNoFileComp=%[5]d
    local shellCompDirectiveFilterFileExt=%[6]d
    local shellCompDirectiveFilterDirs=%[7]d
    local shellCompDirectiveKeepOrder=%[8]d

    local lastParam lastChar flagPrefix requestComp out directive comp lastComp noSpace keepOrder
    local -a completions

    __%[1]s_debug "\n========= starting completion logic =========="
    __%[1]s_debug "CURRENT: ${CURRENT}, words[*]: ${words[*]}"

    # The user could have moved the cursor backwards on the command-line.
    # We need to trigger completion from the $CURRENT location, so we need
    # to truncate the command-line ($words) up to the $CURRENT location.
    # (We cannot use $CURSOR as its value does not work when a command is an alias.)
    words=("${=words[1,CURRENT]}")
    __%[1]s_debug "Truncated words[*]: ${words[*]},"

    lastParam=${words[-1]}
    lastChar=${lastParam[-1]}
    __%[1]s_debug "lastParam: ${lastParam}, lastChar: ${lastChar}"

    # For zsh, when completing a flag with an = (e.g., %[1]s -n=<TAB>)
    # completions must be prefixed with the flag
    setopt local_options BASH_REMATCH
    if [[ "${lastParam}" =~ '-.*=' ]]; then
        # We are dealing with a flag with an =
        flagPrefix="-P ${BASH_REMATCH}"
    fi

    # Prepare the command to obtain completions
    requestComp="${words[1]} %[2]s ${words[2,-1]}"
    if [ "${lastChar}" = "" ]; then
        # If the last parameter is complete (there is a space following it)
        # We add an extra empty parameter so we can indicate this to the go completion code.
        __%[1]s_debug "Adding extra empty parameter"
        requestComp="${requestComp} \"\""
    fi

    __%[1]s_debug "About to call: eval ${requestComp}"

    # Use eval to handle any environment variables and such
    out=$(eval ${requestComp} 2>/dev/null)
    __%[1]s_debug "completion output: ${out}"

    # Extract the directive integer following a : from the last line
    local lastLine
    while IFS='\n' read -r line; do
        lastLine=${line}
    done < <(printf "%%s\n" "${out[@]}")
    __%[1]s_debug "last line: ${lastLine}"

    if [ "${lastLine[1]}" = : ]; then
        directive=${lastLine[2,-1]}
        # Remove the directive including the : and the newline
        local suffix
        (( suffix=${#lastLine}+2))
        out=${out[1,-$suffix]}
    else
        # There is no directive specified.  Leave $out as is.
        __%[1]s_debug "No directive found.  Setting do default"
        directive=0
    fi

    __%[1]s_debug "directive: ${directive}"
    __%[1]s_debug "completions: ${out}"
    __%[1]s_debug "flagPrefix: ${flagPrefix}"

    if [ $((directive & shellCompDirectiveError)) -ne 0 ]; then
        __%[1]s_debug "Completion received error. Ignoring completions."
        return
    fi

    local activeHelpMarker="%[9]s"
    local endIndex=${#activeHelpMarker}
    local startIndex=$((${#activeHelpMarker}+1))
    local hasActiveHelp=0
    while IFS='\n' read -r comp; do
        # Check if this is an activeHelp statement (i.e., prefixed with $activeHelpMarker)
        if [ "${comp[1,$endIndex]}" = "$activeHelpMarker" ];then
            __%[1]s_debug "ActiveHelp found: $comp"
            comp="${comp[$startIndex,-1]}"
            if [ -n "$comp" ]; then
                compadd -x "${comp}"
                __%[1]s_debug "ActiveHelp will need delimiter"
                hasActiveHelp=1
            fi

            continue
        fi

        if [ -n "$comp" ]; then
            # If requested, completions are returned with a description.
            # The description is preceded by a TAB character.
            # For zsh's _describe, we need to use a : instead of a TAB.
            # We first need to escape any : as part of the completion itself.
            comp=${comp//:/\\:}

            local tab="$(printf '\t')"
            comp=${comp//$tab/:}

            __%[1]s_debug "Adding completion: ${comp}"
            completions+=${comp}
            lastComp=$comp
        fi
    done < <(printf "%%s\n" "${out[@]}")

    # Add a delimiter after the activeHelp statements, but only if:
    # - there are completions following the activeHelp statements, or
    # - file completion will be performed (so there will be choices after the activeHelp)
    if [ $hasActiveHelp -eq 1 ]; then
        if [ ${#completions} -ne 0 ] || [ $((directive & shellCompDirectiveNoFileComp)) -eq 0 ]; then
            __%[1]s_debug "Adding activeHelp delimiter"
            compadd -x "--"
            hasActiveHelp=0
        fi
    fi

    if [ $((directive & shellCompDirectiveNoSpace)) -ne 0 ]; then
        __%[1]s_debug "Activating nospace."
        noSpace="-S ''"
    fi

    if [ $((directive & shellCompDirectiveKeepOrder)) -ne 0 ]; then
        __%[1]s_debug "Activating keep order."
        keepOrder="-V"
    fi

    if [ $((directive & shellCompDirectiveFilterFileExt)) -ne 0 ]; then
        # File extension filtering
        local filteringCmd
        filteringCmd='_files'
        for filter in ${completions[@]}; do
            if [ ${filter[1]} != '*' ]; then
                # zsh requires a glob pattern to do file filtering
                filter="\*.$filter"
            fi
            filteringCmd+=" -g $filter"
        done
        filteringCmd+=" ${flagPrefix}"

        __%[1]s_debug "File filtering command: $filteringCmd"
        _arguments '*:filename:'"$filteringCmd"
    elif [ $((directive & shellCompDirectiveFilterDirs)) -ne 0 ]; then
        # File completion for directories only
        local subdir
        subdir="${completions[1]}"
        if [ -n "$subdir" ]; then
            __%[1]s_debug "Listing directories in $subdir"
            pushd "${subdir}" >/dev/null 2>&1
        else
            __%[1]s_debug "Listing directories in ."
        fi

        local result
        _arguments '*:dirname:_files -/'" ${flagPrefix}"
        result=$?
        if [ -n "$subdir" ]; then
            popd >/dev/null 2>&1
        fi
        return $result
    else
        __%[1]s_debug "Calling _describe"
        if eval _describe $keepOrder "completions" completions $flagPrefix $noSpace; then
            __%[1]s_debug "_describe found some completions"

            # Return the success of having called _describe
            return 0
        else
            __%[1]s_debug "_describe did not find completions."
            __%[1]s_debug "Checking if we should do file completion."
            if [ $((directive & shellCompDirectiveNoFileComp)) -ne 0 ]; then
                __%[1]s_debug "deactivating file completion"

                # We must return an error code here to let zsh know that there were no
                # completions found by _describe; this is what will trigger other
                # matching algorithms to attempt to find completions.
                # For example zsh can match letters in the middle of words.
                return 1
            else
                # Perform file completion
                __%[1]s_debug "Activating file completion"

                # We must return the result of this command, so it must be the
                # last command, or else we must store its result to return it.
                _arguments '*:filename:_files'" ${flagPrefix}"
            fi
        fi
    fi
}

# don't run the completion function when being source-ed or eval-ed
if [ "$funcstack[1]" = "_%[1]s" ]; then
    _%[1]s
fi
`, name, compCmd,
		ShellCompDirectiveError, ShellCompDirectiveNoSpace, ShellCompDirectiveNoFileComp,
		ShellCompDirectiveFilterFileExt, ShellCompDirectiveFilterDirs, ShellCompDirectiveKeepOrder,
		activeHelpMarker))
}
