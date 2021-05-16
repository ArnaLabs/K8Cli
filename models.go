package main

type helmRepository struct {
	name     string
	url      string
	username string
	password string
}

type helmRelease struct {
	name       string
	namespace  string
	version    string
	chart      string
	valuesfile string
}

type HelmConfig struct {
	Releases []struct {
		Name       string `yaml:"name"`
		Namespace  string `yaml:"namespace,omitempty"`
		Version    string `yaml:"version,omitempty"`
		Chart      string `yaml:"chart"`
		ValuesFile string `yaml:"values_file,omitempty"`
	} `yaml:"releases"`
	Repositories []struct {
		Name     string `yaml:"name"`
		Url      string `yaml:"url"`
		Username string `yaml:"username,omitempty"`
		Password string `yaml:"password,omitempty"`
	} `yaml:"repositories"`
}

type InitialConfigVals struct {
	ClusterDetails struct {
		ClusterName       string `yaml:"ClusterName"`
		MasterKey         string `yaml:"Masterkey"`
		MasterUrl         string `yaml:"MasterUrl"`
		KubeConfig        string `yaml:"kubeConfig"`
		Configs           string `yaml:"Configs"`
		StorageClassFile  string `yaml:"StorageClassesFile"`
		NameSpaceFile     string `yaml:"NameSpaceFile"`
		ResourceQuotaFile string `yaml:"ResourceQuotaFile"`
	} `yaml:"ClusterDetails"`
}
