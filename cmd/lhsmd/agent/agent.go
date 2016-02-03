/**

Parallel Data Mover is scalable system to copy or migrate data between
various storage systems. It supports multliple types of sources and
destinations, including POSIX, S3, HPSS, etc.

Use cases include:
  * Data movement for Lustre HSM.
  * Offsite replication for DR
  * Lustre file-level replication
  * Storage rebalancing within a single tier
  * Migration between filesytems (e.g GPFS - > Lustre)

Initially the main focus is for HSM.
*/

package agent

import (
	"fmt"
	"os/exec"
	"sync"

	"github.intel.com/hpdd/liblog"
	"github.intel.com/hpdd/lustre/fs"
	"github.intel.com/hpdd/lustre/hsm"
	"github.intel.com/hpdd/policy/pkg/client"
	"github.intel.com/hpdd/svclog"
)

type (
	backendMap map[ArchiveID]*exec.Cmd

	// HsmAgent for a single filesytem and a collection of backends.
	HsmAgent struct {
		config    *Config
		client    *client.Client
		wg        sync.WaitGroup
		Endpoints *Endpoints
		Backends  backendMap
		mu        sync.Mutex // Protject the agent
		agent     hsm.Agent
	}

	// Transport for backend plugins
	Transport interface {
		Init(*Config, *HsmAgent) error
	}
)

// New accepts a config and returns a *HsmAgent
func New(cfg *Config) (*HsmAgent, error) {
	client, err := client.New(cfg.Lustre.String())
	if err != nil {
		return nil, err
	}

	ct := &HsmAgent{
		config:    cfg,
		client:    client,
		Endpoints: NewEndpoints(),
		Backends:  make(backendMap),
	}

	return ct, nil
}

// Start backgrounds the agent and starts backend data movers
func (ct *HsmAgent) Start() error {
	for _, t := range transports {
		if err := t.Init(ct.config, ct); err != nil {
			return err
		}
	}

	if err := ct.initAgent(); err != nil {
		return err
	}

	for i := 0; i < ct.config.Processes; i++ {
		ct.addHandler(fmt.Sprintf("handler-%d", i))
	}

	for id, pluginConf := range ct.config.Archives {
		cmd, err := startPlugin(id, pluginConf)
		if err != nil {
			return err
		}
		ct.Backends[id] = cmd

		svclog.Debug("Started %s (PID: %d) for archive #%d", cmd.Path, cmd.Process.Pid, id)
	}

	// TODO: Monitor backends to restart any that die

	ct.wg.Wait()
	return nil
}

// Stop shuts down all backend data movers and kills the agent
func (ct *HsmAgent) Stop() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if ct.agent != nil {
		ct.agent.Stop()
	}
}

// Root returns a fs.RootDir representing the Lustre filesystem root
func (ct *HsmAgent) Root() fs.RootDir {
	return ct.client.Root()
}

func (ct *HsmAgent) initAgent() (err error) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.agent, err = hsm.Start(ct.client.Root())
	return
}

func (ct *HsmAgent) handleActions(tag string) {
	ch := ct.agent.Actions()
	for ai := range ch {
		liblog.Debug("%s: incoming: %s", tag, ai)
		aih, err := ai.Begin(0, false)
		if err != nil {
			liblog.Debug("%s: begin failed: %v", tag, err)
			continue
		}

		if e, ok := ct.Endpoints.Get(uint32(aih.ArchiveID())); ok {
			e.Send(aih)
		} else {
			liblog.Debug("%s: no handler for archive %d", tag, aih.ArchiveID())
			aih.End(0, 0, 0, -1)
		}
	}
}

func (ct *HsmAgent) addHandler(tag string) {
	ct.wg.Add(1)
	go func() {
		ct.handleActions(tag)
		ct.wg.Done()
	}()
}

var transports []Transport

// RegisterTransport registers the transport in the list of known transports
func RegisterTransport(t Transport) {
	transports = append(transports, t)
}