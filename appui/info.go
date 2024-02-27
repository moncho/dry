package appui

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/go-units"
)

type infoRenderer struct {
	env system.Info
}

// NewDockerInfoRenderer creates renderer for for docker info
func NewDockerInfoRenderer(env system.Info) fmt.Stringer {
	return &infoRenderer{
		env: env,
	}
}

// Render system-wide information
func (r *infoRenderer) String() string {

	buffer := new(bytes.Buffer)
	info := r.env

	writeKV(buffer, "Containers", info.Containers)
	writeKV(buffer, " Running", info.ContainersRunning)
	writeKV(buffer, " Paused", info.ContainersPaused)
	writeKV(buffer, " Stopped", info.ContainersStopped)
	writeKV(buffer, "Images", info.Images)
	writeKVIfNotEmpty(buffer, "Server Version", info.ServerVersion)
	writeKVIfNotEmpty(buffer, "Storage Driver", info.Driver)
	if info.DriverStatus != nil {
		for _, pair := range info.DriverStatus {
			writeKV(buffer, pair[0], pair[1])

			// print a warning if devicemapper is using a loopback file
			if pair[0] == "Data loop file" {
				buffer.WriteString(" WARNING: Usage of loopback devices is strongly discouraged for production use. Use `--storage-opt dm.thinpooldev` to specify a custom block storage device.")
			}
		}

	}
	if info.SystemStatus != nil {
		for _, pair := range info.SystemStatus {
			writeKV(buffer, pair[0], pair[1])
		}
	}
	writeKVIfNotEmpty(buffer, "Logging Driver", info.LoggingDriver)
	writeKVIfNotEmpty(buffer, "Cgroup Driver", info.CgroupDriver)

	buffer.WriteString("<white> Plugins:</>\n")
	writeKV(buffer, " Volume", strings.Join(info.Plugins.Volume, " "))
	writeKV(buffer, " Network", strings.Join(info.Plugins.Network, " "))

	if len(info.Plugins.Authorization) != 0 {
		buffer.WriteString("<white> Authorization:</>")
		buffer.WriteString(strings.Join(info.Plugins.Authorization, " "))
		buffer.WriteString("")
	}

	writeKV(buffer, "Swarm", info.Swarm.LocalNodeState)
	if info.Swarm.LocalNodeState != swarm.LocalNodeStateInactive && info.Swarm.LocalNodeState != swarm.LocalNodeStateLocked {
		writeKV(buffer, " NodeID", info.Swarm.NodeID)
		if info.Swarm.Error != "" {
			writeKV(buffer, " Error", info.Swarm.Error)
		}
		writeKV(buffer, " Is Manager", info.Swarm.ControlAvailable)
		if info.Swarm.ControlAvailable && info.Swarm.Error == "" && info.Swarm.LocalNodeState != swarm.LocalNodeStateError {
			writeKV(buffer, " ClusterID", info.Swarm.Cluster.ID)
			writeKV(buffer, " Managers", info.Swarm.Managers)
			writeKV(buffer, " Nodes", info.Swarm.Nodes)
			buffer.WriteString("<white> Orchestration:</>")
			taskHistoryRetentionLimit := int64(0)
			if info.Swarm.Cluster.Spec.Orchestration.TaskHistoryRetentionLimit != nil {
				taskHistoryRetentionLimit = *info.Swarm.Cluster.Spec.Orchestration.TaskHistoryRetentionLimit
			}
			writeKV(buffer, "  Task History Retention Limit", taskHistoryRetentionLimit)
			buffer.WriteString("<white> Raft:</>")
			writeKV(buffer, "  Snapshot Interval", info.Swarm.Cluster.Spec.Raft.SnapshotInterval)
			if info.Swarm.Cluster.Spec.Raft.KeepOldSnapshots != nil {
				writeKV(buffer, "  Number of Old Snapshots to Retain", *info.Swarm.Cluster.Spec.Raft.KeepOldSnapshots)
			}
			writeKV(buffer, "  Heartbeat Tick", info.Swarm.Cluster.Spec.Raft.HeartbeatTick)
			writeKV(buffer, "  Election Tick", info.Swarm.Cluster.Spec.Raft.ElectionTick)
			buffer.WriteString("<white> Dispatcher:</>")

			writeKV(buffer, "  Heartbeat Period", units.HumanDuration(info.Swarm.Cluster.Spec.Dispatcher.HeartbeatPeriod))
			buffer.WriteString("<white> CA Configuration:</>")
			writeKV(buffer, "  Expiry Duration", units.HumanDuration(info.Swarm.Cluster.Spec.CAConfig.NodeCertExpiry))
			if len(info.Swarm.Cluster.Spec.CAConfig.ExternalCAs) > 0 {
				buffer.WriteString("<white> External CAs:</>")

				for _, entry := range info.Swarm.Cluster.Spec.CAConfig.ExternalCAs {
					writeKV(buffer, string(entry.Protocol), entry.URL)
				}
			}
		}
		writeKV(buffer, " Node Address", info.Swarm.NodeAddr)
		managers := []string{}
		for _, entry := range info.Swarm.RemoteManagers {
			managers = append(managers, entry.Addr)
		}
		if len(managers) > 0 {
			sort.Strings(managers)
			buffer.WriteString("<white> Manager Addresses:</>")

			for _, entry := range managers {
				writeKV(buffer, "  %s", entry)
			}
		}
	}

	if len(info.Runtimes) > 0 {
		buffer.WriteString("<white> Runtimes:</>")
		for name, runtime := range info.Runtimes {
			writeKV(buffer, name, runtime.Path)
		}
	}
	writeKV(buffer, "Default Runtime", info.DefaultRuntime)

	if info.OSType == "linux" {
		writeKV(buffer, "Init Binary", info.InitBinary)

		for _, ci := range []struct {
			name   string
			commit system.Commit
		}{
			{"containerd", info.ContainerdCommit},
			{"runc", info.RuncCommit},
			{"init", info.InitCommit},
		} {
			writeKV(buffer, fmt.Sprintf("%s version", ci.name), ci.commit.ID)
			if ci.commit.ID != ci.commit.Expected {
				writeKV(buffer, " (expected)", ci.commit.Expected)
			}
			buffer.WriteString("")
		}
		if len(info.SecurityOptions) != 0 {
			kvs, err := system.DecodeSecurityOptions(info.SecurityOptions)
			if err == nil {
				buffer.WriteString("<white> Security Options:</>\n")
				for _, so := range kvs {
					buffer.WriteString(fmt.Sprintf("<white>  * %s:</>\n", so.Name))
					for _, o := range so.Options {
						switch o.Key {
						case "profile":
							if o.Value != "default" {
								buffer.WriteString("  WARNING: You're not using the default seccomp profile")
							}
							writeKV(buffer, "   Profile", o.Value)
						}
					}
				}
			}
		}
	}

	// Isolation only has meaning on a Windows daemon.
	if info.OSType == "windows" {
		writeKV(buffer, "Default Isolation", info.Isolation)
	}

	writeKVIfNotEmpty(buffer, "Kernel Version", info.KernelVersion)
	writeKVIfNotEmpty(buffer, "Operating System", info.OperatingSystem)
	writeKVIfNotEmpty(buffer, "OSType", info.OSType)
	writeKVIfNotEmpty(buffer, "Architecture", info.Architecture)
	writeKV(buffer, "CPUs", info.NCPU)
	writeKV(buffer, "Total Memory", units.BytesSize(float64(info.MemTotal)))
	writeKVIfNotEmpty(buffer, "Name", info.Name)
	writeKVIfNotEmpty(buffer, "ID", info.ID)
	writeKV(buffer, "Docker Root Dir", info.DockerRootDir)
	writeKV(buffer, "Debug Mode (server)", info.Debug)

	if info.Debug {
		writeKV(buffer, " File Descriptors", info.NFd)
		writeKV(buffer, " Goroutines", info.NGoroutines)
		writeKV(buffer, " System Time", info.SystemTime)
		writeKV(buffer, " EventsListeners", info.NEventsListener)
	}

	writeKVIfNotEmpty(buffer, "Http Proxy", info.HTTPProxy)
	writeKVIfNotEmpty(buffer, "Https Proxy", info.HTTPSProxy)
	writeKVIfNotEmpty(buffer, "No Proxy", info.NoProxy)

	if info.IndexServerAddress != "" {
		//TODO read username from config
		/*u := dockerCli.ConfigFile().AuthConfigs[info.IndexServerAddress].Username
		if len(u) > 0 {
			writeKV(buffer, "Username", u)
		}*/
		writeKV(buffer, "Registry", info.IndexServerAddress)
	}

	// Only output these warnings if the server does not support these features
	if info.OSType != "windows" {
		if !info.MemoryLimit {
			buffer.WriteString("WARNING: No memory limit support")
		}
		if !info.SwapLimit {
			buffer.WriteString("WARNING: No swap limit support")
		}
		if !info.KernelMemory {
			buffer.WriteString("WARNING: No kernel memory limit support")
		}
		if !info.OomKillDisable {
			buffer.WriteString("WARNING: No oom kill disable support")
		}
		if !info.CPUCfsQuota {
			buffer.WriteString("WARNING: No cpu cfs quota support")
		}
		if !info.CPUCfsPeriod {
			buffer.WriteString("WARNING: No cpu cfs period support")
		}
		if !info.CPUShares {
			buffer.WriteString("WARNING: No cpu shares support")
		}
		if !info.CPUSet {
			buffer.WriteString("WARNING: No cpuset support")
		}
		if !info.IPv4Forwarding {
			buffer.WriteString("WARNING: IPv4 forwarding is disabled")
		}
		if !info.BridgeNfIptables {
			buffer.WriteString("WARNING: bridge-nf-call-iptables is disabled")
		}
		if !info.BridgeNfIP6tables {
			buffer.WriteString("WARNING: bridge-nf-call-ip6tables is disabled")
		}
	}

	if info.Labels != nil {
		buffer.WriteString("<white>Labels:</>")
		for _, attribute := range info.Labels {
			writeKV(buffer, " %s", attribute)
		}
		// TODO: Engine labels with duplicate keys has been deprecated in 1.13 and will be error out
		// after 3 release cycles (1.16). For now, a WARNING will be generated. The following will
		// be removed eventually.
		labelMap := map[string]string{}
		for _, label := range info.Labels {
			stringSlice := strings.SplitN(label, "=", 2)
			if len(stringSlice) > 1 {
				// If there is a conflict we will throw out a warning
				if v, ok := labelMap[stringSlice[0]]; ok && v != stringSlice[1] {
					buffer.WriteString("WARNING: labels with duplicate keys and conflicting values have been deprecated")
					break
				}
				labelMap[stringSlice[0]] = stringSlice[1]
			}
		}
	}

	writeKV(buffer, "Experimental", info.ExperimentalBuild)
	if info.RegistryConfig != nil && (len(info.RegistryConfig.InsecureRegistryCIDRs) > 0 || len(info.RegistryConfig.IndexConfigs) > 0) {
		buffer.WriteString("<white> Insecure Registries:</>\n")
		for _, registry := range info.RegistryConfig.IndexConfigs {
			if !registry.Secure {
				buffer.WriteString(fmt.Sprintf("  %s\n", registry.Name))
			}
		}

		for _, registry := range info.RegistryConfig.InsecureRegistryCIDRs {
			mask, _ := registry.Mask.Size()
			buffer.WriteString(fmt.Sprintf("  %s/%d\n", registry.IP.String(), mask))
		}
	}

	if info.RegistryConfig != nil && len(info.RegistryConfig.Mirrors) > 0 {
		writeKV(buffer, "Registry Mirrors", strings.Join(info.RegistryConfig.Mirrors, " "))
	}

	writeKV(buffer, "Live Restore Enabled", info.LiveRestoreEnabled)
	return buffer.String()
}

func writeKVIfNotEmpty(buffer *bytes.Buffer, key string, value interface{}) {
	if value != nil {
		writeKV(buffer, key, value)
	}
}

// writeKV write into the given buffer "key: value"
func writeKV(buffer *bytes.Buffer, key string, value interface{}) {
	buffer.WriteString(fmt.Sprintf("<white> %s </>: %v\n", key, value))
}
