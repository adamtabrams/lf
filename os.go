// +build !windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	envOpener = os.Getenv("OPENER")
	envEditor = os.Getenv("EDITOR")
	envPager  = os.Getenv("PAGER")
	envShell  = os.Getenv("SHELL")
)

var (
	gDefaultShell      = "sh"
	gDefaultSocketProt = "unix"
	gDefaultSocketPath string
)

var (
	gUser        *user.User
	gConfigPaths []string
	gMarksPath   string
	gHistoryPath string
)

func init() {
	if envOpener == "" {
		if runtime.GOOS == "darwin" {
			envOpener = "open"
		} else {
			envOpener = "xdg-open"
		}
	}

	if envEditor == "" {
		envEditor = "vi"
	}

	if envPager == "" {
		envPager = "less"
	}

	if envShell == "" {
		envShell = "sh"
	}

	u, err := user.Current()
	if err != nil {
		log.Printf("user: %s", err)
		if os.Getenv("HOME") == "" {
			log.Print("$HOME variable is empty or not set")
		}
		if os.Getenv("USER") == "" {
			log.Print("$USER variable is empty or not set")
		}
	}
	gUser = u

	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(gUser.HomeDir, ".config")
	}

	gConfigPaths = []string{
		filepath.Join("/etc", "lf", "lfrc"),
		filepath.Join(config, "lf", "lfrc"),
	}

	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(gUser.HomeDir, ".local", "share")
	}

	gMarksPath = filepath.Join(data, "lf", "marks")
	gHistoryPath = filepath.Join(data, "lf", "history")

	gDefaultSocketPath = filepath.Join(os.TempDir(), fmt.Sprintf("lf.%s.sock", gUser.Username))
}

func detachedCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func pauseCommand() *exec.Cmd {
	cmd := `echo
	        echo -n 'Press any key to continue'
	        old=$(stty -g)
	        stty raw -echo
	        eval "ignore=\$(dd bs=1 count=1 2> /dev/null)"
	        stty $old
	        echo`

	return exec.Command(gOpts.shell, "-c", cmd)
}

func shellCommand(s string, args []string) *exec.Cmd {
	if len(gOpts.ifs) != 0 {
		s = fmt.Sprintf("IFS='%s'; %s", gOpts.ifs, s)
	}

	args = append([]string{"-c", s, "--"}, args...)

	args = append(gOpts.shellopts, args...)

	return exec.Command(gOpts.shell, args...)
}

func setDefaults() {
	gOpts.cmds["open"] = &execExpr{"&", `$OPENER "$f"`}
	gOpts.keys["e"] = &execExpr{"$", `$EDITOR "$f"`}
	gOpts.keys["i"] = &execExpr{"$", `$PAGER "$f"`}
	gOpts.keys["w"] = &execExpr{"$", "$SHELL"}

	gOpts.cmds["doc"] = &execExpr{"$", "lf -doc | $PAGER"}
	gOpts.keys["<f-1>"] = &callExpr{"doc", nil, 1}
}

func moveCursor(y, x int) {
	fmt.Printf("\033[%d;%dH", y, x)
}

func isExecutable(f os.FileInfo) bool {
	return f.Mode()&0111 != 0
}

func isHidden(f os.FileInfo, path string) (hidden bool) {
	for _, pattern := range gOpts.hiddenFiles {
		if len(pattern) == 0 {
			continue
		}

		matched := matchPattern(strings.TrimPrefix(pattern, "!"), f.Name(), path)
		if pattern[0] == '!' && matched {
			hidden = false
		} else if matched {
			hidden = true
		}
	}
	return hidden
}

func matchPattern(pattern, name, path string) bool {
	matchStr := name

	if pattern[0] == '~' {
		pattern = gUser.HomeDir + strings.TrimPrefix(pattern, "~")
	}
	if pattern[0] == filepath.Separator {
		matchStr = filepath.Join(path, name)
	}

	isMatch, err := filepath.Match(pattern, matchStr)
	if err != nil {
		log.Printf("pattern %s caused error with %s", pattern, matchStr)
		return false
	}
	return isMatch
}

func exportFiles(f string, fs []string) {
	envFile := f
	envFiles := strings.Join(fs, gOpts.filesep)

	os.Setenv("f", envFile)
	os.Setenv("fs", envFiles)

	if len(fs) == 0 {
		os.Setenv("fx", envFile)
	} else {
		os.Setenv("fx", envFiles)
	}
}
