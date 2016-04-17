package appui

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	dockerTypes "github.com/docker/engine-api/types"
	"github.com/docker/go-units"
	"github.com/moncho/dry/ui"
)

type infoRenderer struct {
	env dockerTypes.Info
}

//NewDockerInfoRenderer creates renderer for for docker info
func NewDockerInfoRenderer(env dockerTypes.Info) ui.Renderer {
	return &infoRenderer{
		env: env,
	}
}

//Render system-wide information
func (r *infoRenderer) Render() string {
	buffer := new(bytes.Buffer)
	info := r.env

	writeKV(buffer, "Containers", info.Containers)
	writeKV(buffer, " Running", info.ContainersRunning)
	writeKV(buffer, " Paused", info.ContainersPaused)
	writeKV(buffer, " Stopped", info.ContainersStopped)
	writeKV(buffer, "Images", info.Images)
	writeKV(buffer, "Server Version", info.ServerVersion)
	writeKV(buffer, "Storage Driver", info.Driver)
	if info.DriverStatus != nil {
		for _, pair := range info.DriverStatus {
			writeKV(buffer, pair[0], pair[1])

			// print a warning if devicemapper is using a loopback file
			if pair[0] == "Data loop file" {
				buffer.WriteString(" WARNING: Usage of loopback devices is strongly discouraged for production use. Either use `--storage-opt dm.thinpooldev` or use `--storage-opt dm.no_warn_on_loop_devices=true` to suppress this warning.\n")
			}
		}

	}
	if info.SystemStatus != nil {
		for _, pair := range info.SystemStatus {
			writeKV(buffer, pair[0], pair[1])
		}
	}
	writeKV(buffer, "Execution Driver", info.ExecutionDriver)
	writeKV(buffer, "Logging Driver", info.LoggingDriver)
	writeKV(buffer, "Cgroup Driver", info.CgroupDriver)

	buffer.WriteString("<white>Plugins:</> \n")
	writeKV(buffer, " Volume", strings.Join(info.Plugins.Volume, " "))
	writeKV(buffer, " Network", strings.Join(info.Plugins.Network, " "))

	if len(info.Plugins.Authorization) != 0 {
		writeKV(buffer, " Authorization", strings.Join(info.Plugins.Authorization, " "))
	}

	writeKV(buffer, "Kernel Version", info.KernelVersion)
	writeKV(buffer, "Operating System", info.OperatingSystem)
	writeKV(buffer, "OSType", info.OSType)
	writeKV(buffer, "Architecture", info.Architecture)
	writeKV(buffer, "CPUs", info.NCPU)
	writeKV(buffer, "Total Memory", units.BytesSize(float64(info.MemTotal)))
	writeKV(buffer, "Name", info.Name)
	writeKV(buffer, "ID", info.ID)
	writeKV(buffer, "Docker Root Dir", info.DockerRootDir)
	writeKV(buffer, "Debug Mode (client)", isDebugEnabled())
	writeKV(buffer, "Debug Mode (server)", info.Debug)

	if info.Debug {
		writeKV(buffer, " File Descriptors", info.NFd)
		writeKV(buffer, " Goroutines", info.NGoroutines)
		writeKV(buffer, " System Time", info.SystemTime)
		writeKV(buffer, " EventsListeners", info.NEventsListener)
	}

	writeKV(buffer, "Http Proxy", info.HTTPProxy)
	writeKV(buffer, "Https Proxy", info.HTTPSProxy)
	writeKV(buffer, "No Proxy", info.NoProxy)
	if info.IndexServerAddress != "" {
		/*u := cli.configFile.AuthConfigs[info.IndexServerAddress].Username
		if len(u) > 0 {
			writeKV(buffer, "Username", u)
		}*/
		writeKV(buffer, "Registry", info.IndexServerAddress)
	}

	// Only output these warnings if the server does not support these features
	if info.OSType != "windows" {
		if !info.MemoryLimit {
			buffer.WriteString("WARNING: No memory limit support\n")
		}
		if !info.SwapLimit {
			buffer.WriteString("WARNING: No swap limit support\n")
		}
		if !info.KernelMemory {
			buffer.WriteString("WARNING: No kernel memory limit support\n")
		}
		if !info.OomKillDisable {
			buffer.WriteString("WARNING: No oom kill disable support\n")
		}

		if !info.CPUCfsQuota {
			buffer.WriteString("WARNING: No cpu cfs quota support\n")
		}
		if !info.CPUCfsPeriod {
			buffer.WriteString("WARNING: No cpu cfs period support\n")
		}
		if !info.CPUShares {
			buffer.WriteString("WARNING: No cpu shares support\n")
		}
		if !info.CPUSet {
			buffer.WriteString("WARNING: No cpuset support\n")
		}
		if !info.IPv4Forwarding {
			buffer.WriteString("WARNING: IPv4 forwarding is disabled\n")
		}
		if !info.BridgeNfIptables {
			buffer.WriteString("WARNING: bridge-nf-call-iptables is disabled\n")
		}
		if !info.BridgeNfIP6tables {
			buffer.WriteString("WARNING: bridge-nf-call-ip6tables is disabled\n")
		}
	}
	if info.Labels != nil {
		buffer.WriteString("<white>Labels:</>\n")
		for _, attribute := range info.Labels {
			separator := strings.Index(attribute, "=")
			if separator > 0 {
				writeKV(buffer, " "+attribute[0:separator], attribute[separator+1:])
			} else {
				buffer.WriteString(fmt.Sprintf("<white>%s</>\n", attribute))
			}
		}
	}

	writeKV(buffer, "Experimental", info.ExperimentalBuild)
	if info.ClusterStore != "" {
		writeKV(buffer, "Cluster Store", info.ClusterStore)
	}

	if info.ClusterAdvertise != "" {
		writeKV(buffer, "Cluster Advertise", info.ClusterAdvertise)
	}
	if info.RegistryConfig != nil && (len(info.RegistryConfig.InsecureRegistryCIDRs) > 0 || len(info.RegistryConfig.IndexConfigs) > 0) {
		buffer.WriteString("<white>Insecure registries:</>\n")
		for _, registry := range info.RegistryConfig.IndexConfigs {
			if registry.Secure == false {
				buffer.WriteString(fmt.Sprintf(" %s\n", registry.Name))
			}
		}

		for _, registry := range info.RegistryConfig.InsecureRegistryCIDRs {
			mask, _ := registry.Mask.Size()
			writeKV(buffer, registry.IP.String(), mask)
		}
	}
	return buffer.String()
}

//writeKV write into the given buffer "key: value"
func writeKV(buffer *bytes.Buffer, key string, value interface{}) {
	buffer.WriteString(fmt.Sprintf("<white> %s </>: %v\n", key, value))
}

// isDebugEnabled checks whether the debug flag is set or not.
func isDebugEnabled() bool {
	return os.Getenv("DEBUG") != ""
}
