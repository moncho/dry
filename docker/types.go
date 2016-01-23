package docker

// Version contains response of Remote API:
// GET "/version"
//Copied from docker/engine-api/types until docker library is fully replaced
type Version struct {
	Version       string
	APIVersion    string
	GitCommit     string
	GoVersion     string
	Os            string
	Arch          string
	KernelVersion string
	Experimental  bool
	BuildTime     string
}
