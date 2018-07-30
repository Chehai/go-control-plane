package common

type Endpoint struct {
	Host string
	Port uint32
}

type Cluster struct {
	Name      string
	Endpoints []Endpoint
}
