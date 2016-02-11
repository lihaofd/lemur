package main

import (
	"flag"

	"github.intel.com/hpdd/logging/alert"
	"github.intel.com/hpdd/policy/pdm/dmplugin"
)

var (
	archive      uint
	agentAddress string
)

func init() {
	flag.UintVar(&archive, "archive", 1, "archive id")
	flag.StringVar(&agentAddress, "agent", ":4242", "Lustre client mountpoint")
}

type Mover struct {
	fsName    string
	archiveID uint32
}

func (m *Mover) FsName() string {
	return m.fsName
}

func (m *Mover) ArchiveID() uint32 {
	return m.archiveID
}

func noop(agentAddress string) {
	done := make(chan struct{})

	plugin, err := dmplugin.New(agentAddress)
	if err != nil {
		alert.Fatal(err)
	}
	mover := Mover{fsName: "noop", archiveID: uint32(archive)}
	plugin.AddMover(&mover)

	<-done
	plugin.Stop()
}

func main() {
	flag.Parse()

	noop(agentAddress)
}
