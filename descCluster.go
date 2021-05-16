package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type kubeConfig struct {
	Apiversion string `json:"apiVersion"`
	Clusters   []struct {
		Cluster struct {
			CertificateAuthorityData string `json:"certificate-authority-data"`
			Server                   string `json:"server"`
		} `json:"cluster"`
		Name string `json:"name"`
	} `json:"clusters"`
	Contexts []struct {
		Context struct {
			Cluster string `json:"cluster"`
			User    string `json:"user"`
		} `json:"context"`
		Name string `json:"name"`
	} `json:"contexts"`
	CurrentContext string `json:"current-context"`
	Kind           string `json:"kind"`
	Preferences    struct {
	} `json:"preferences"`
	Users []struct {
		Name string `json:"name"`
		User struct {
			Exec struct {
				Apiversion string   `json:"apiVersion"`
				Args       []string `json:"args"`
				Command    string   `json:"command"`
				Env        []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"env"`
			} `json:"exec"`
		} `json:"user"`
	} `json:"users"`
}


func getClusterEndpoint(context string, kubeConfigFile string ) string {
	var config kubeConfig
	var configCluster, endpoint string

	fmt.Println("kubeConfigFile:", kubeConfigFile)

	yamlFile, err := ioutil.ReadFile(kubeConfigFile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}

	j := 0
	for range config.Contexts {
		if config.Contexts[j].Name == context {
			configCluster = config.Contexts[j].Context.Cluster
			break
		}
		j++
	}

	k := 0
	for range config.Clusters {
		if config.Clusters[j].Name == configCluster {
			endpoint = config.Clusters[j].Cluster.Server
			break
		}
		k++
	}

	return endpoint
}