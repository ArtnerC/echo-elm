package resolver_test

import "os"

func writeAll(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
