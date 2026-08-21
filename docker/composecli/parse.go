package composecli

import "strings"

// parseConfigHashes reads the output of `compose config --hash='*'`, which
// prints one `<service> <hash>` line per service.
func parseConfigHashes(out string) map[string]string {
	hashes := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		service, hash, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok || service == "" || hash == "" {
			continue
		}
		hashes[service] = strings.TrimSpace(hash)
	}
	return hashes
}
