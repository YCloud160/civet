package civet

import "fmt"

type Endpoint struct {
	IP     string
	Port   string
	Weight int32
}

func (e *Endpoint) IPPort() string {
	return fmt.Sprintf("%s:%s", e.IP, e.Port)
}
