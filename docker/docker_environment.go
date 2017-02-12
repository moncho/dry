package docker

//DockerEnv holds Docker-related environment variables
type DockerEnv struct {
	DockerHost       string
	DockerTLSVerify  bool //tls must be verified
	DockerCertPath   string
	DockerAPIVersion string
}

//NewEnv creates a new docker environment struct
func NewEnv() *DockerEnv {
	return &DockerEnv{DockerAPIVersion: "1.25"}
}
