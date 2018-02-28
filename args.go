package main

import (
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type Commands struct {
	before     []*exec.Cmd         // list of commands to run BEFORE anything else
	run        []*exec.Cmd         // list of commands to run BEFORE the primar
	start      []*exec.Cmd         // list of services to start
	credential *syscall.Credential // credentials for primary command
}

//
// Removes --before and --start and --run commands options and arguments from os.Args
// Removes --user <uid|username> options and applies the credentials to following
//           start or run commands and primary command
// Returns array of removed run commands, and an array of removed start commands
//
func removeCommandsFromOsArgs() Commands {

	var newOsArgs = []string{}
	var commands = Commands{}

	var cmd *exec.Cmd
	var cmd_user *user.User

    if debugFlag {
        log.Printf("")
        log.Printf("dockerfy args BEFORE removing commands:\n")

        for i := 0; i < len(os.Args); i++ {
            log.Printf("\t%d: %s", i, os.Args[i])
        }
    }

	for i := 0; i < len(os.Args); i++ {
        // docker-compose.yml files are buggy on \ continuation characters
        arg_i := strings.TrimSpace(os.Args[i])
		switch {

		case ("--before" == arg_i || "-before" == arg_i) && cmd == nil:
			cmd = &exec.Cmd{Stdout: os.Stdout,
				Stderr:      os.Stderr,
				SysProcAttr: &syscall.SysProcAttr{Credential: commands.credential}}
			commands.before = append(commands.before, cmd)

		case ("--start" == arg_i || "-start" == arg_i) && cmd == nil:
			cmd = &exec.Cmd{Stdout: os.Stdout,
				Stderr:      os.Stderr,
				SysProcAttr: &syscall.SysProcAttr{Credential: commands.credential}}
			commands.start = append(commands.start, cmd)

		case ("--run" == arg_i || "-run" == arg_i) && cmd == nil:
			cmd = &exec.Cmd{Stdout: os.Stdout,
				Stderr:      os.Stderr,
				SysProcAttr: &syscall.SysProcAttr{Credential: commands.credential}}
			commands.run = append(commands.run, cmd)

		case ("--user" == arg_i || "-user" == arg_i) && cmd == nil:
			if os.Getuid() != 0 {
				log.Fatalf("dockerfy must run as root to use the --user flag")
			}
			cmd_user = &user.User{}

		case "--" == arg_i && cmd != nil: // End of args for this cmd
			cmd = nil

		default:
			if cmd_user != nil {
				// Expect a username or uid
				var err1 error

				user_name_or_id := arg_i
				cmd_user, err1 = user.LookupId(user_name_or_id)
				if cmd_user == nil {
					// Not a userid, try as a username
					cmd_user, err1 = user.Lookup(user_name_or_id)
					if cmd_user == nil {
						log.Fatalf("unknown user: '%s': %s", user_name_or_id, err1)
					}
				}
				uid, _ := strconv.Atoi(cmd_user.Uid)
				gid, _ := strconv.Atoi(cmd_user.Gid)

				commands.credential = new(syscall.Credential)
				commands.credential.Uid = uint32(uid)
				commands.credential.Gid = uint32(gid)

				cmd_user = nil
			} else if cmd != nil {
				// Expect a command first, then a series of arguments
				if len(cmd.Path) == 0 {
					cmd.Path = arg_i
					if filepath.Base(cmd.Path) == cmd.Path {
						cmd.Path, _ = exec.LookPath(cmd.Path)
					}
				}
        // Only trim our own args, not --before cmd's or not --run cmd's or --start cmd's
				cmd.Args = append(cmd.Args, os.Args[i])
			} else {
				newOsArgs = append(newOsArgs, arg_i)
			}
		}
	}
	if cmd_user != nil {
		log.Fatalln("need a username or uid after the --user flag")
	}
	if cmd != nil {
		log.Fatalf("need a command after the --before or --start or --run flag")
	}
	os.Args = newOsArgs

    if debugFlag {
        log.Printf("")
        log.Printf("dockerfy args AFTER removing commands:\n")

        for i := 0; i < len(os.Args); i++ {
            log.Printf("\t%d: %s", i, os.Args[i])
        }
    }

	return commands
}

func toString(cmd *exec.Cmd) string {
	s := ""
	for _, arg := range cmd.Args {
		s += arg + " "
	}
	return strings.TrimSpace(s)
}
