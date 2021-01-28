package util

import (
	"os/user"
	"strconv"
	"syscall"
)

// GetSyscallCredential returns the `syscall.Credential` structure of the
// given system `username`.
func GetSyscallCredential(username string) (*syscall.Credential, error) {

	u, err := user.Lookup(username)
	if err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return nil, err
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return nil, err
	}

	c := syscall.Credential{
		Uid:         uint32(uid),
		Gid:         uint32(gid),
		NoSetGroups: true,
	}

	return &c, nil
}
