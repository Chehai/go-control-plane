package common

type Endpoint struct {
	Host string `json:"host"`
	Port uint32 `json:"port"`
}

type Cluster struct {
	Name      string `json:"name"`
	Endpoints []Endpoint `json:"endpoints"`
}

type Route struct {
	Vhost string
	Cluster string
}
