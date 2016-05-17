package docker

//DockerEnv holds the Docker-related environment variables defined
type DockerEnv struct {
	DockerHost       string
	DockerTLSVerify  bool //tls must be verified
	DockerCertPath   string
	DockerAPIVersion string
}
