package manageCluster

import "C"
import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
	"text/template"
)

const storageClassSuffix string = ".storageclass.storage.k8s.io/"

type NameSpaceVals struct {
	Delete struct {
		Enable bool `yaml:"Enable"`
	}
	NameSpace []struct {
		Name          string            `yaml:"Name"`
		ResourceQuota string            `yaml:"ResourceQuota"`
		DefaultQuota  string            `yaml:"DefaultQuota"`
		Labels        map[string]string `yaml:"Labels"`
	} `yaml:"NameSpace"`
}
type ResourceQuotaVals struct {
	ResourceQuota []struct {
		ResourceQuotaName        string `yaml:"QuotaName"`
		RequestsCPU              string `yaml:"RequestsCPU"`
		LimitsCPU                string `yaml:"LimitsCPU"`
		RequestsMemory           string `yaml:"RequestsMemory"`
		LimitsMemory             string `yaml:"LimitsMemory"`
		Pods                     string `yaml:"Pods"`
		RequestsStorage          string `yaml:"Name"`
		RequestsEphemeralStorage string `yaml:"RequestsStorage"`
		LimitsEphemeralStorage   string `yaml:"LimitsEphemeralStorage"`
		StorageClasses           []struct {
			Name            string `yaml:"Name"`
			RequestsStorage string `yaml:"RequestsStorage"`
		} `yaml:"StorageClasses"`
		Labels map[string]string `yaml:"Labels"`
	} `yaml:"ResourceQuota"`
}
type StorageClassVals struct {
	Delete struct {
		Enable bool `yaml:"Enable"`
	} `yaml:"Delete"`
	StorageClasses []struct {
		Name              string            `yaml:"Name"`
		Provisioner       string            `yaml:"Provisioner"`
		Parameters        map[string]string `yaml:"Parameters"`
		ReclaimPolicy     string            `yaml:"ReclaimPolicy"`
		VolumeBindingMode string            `yaml:"VolumeBindingMode"`
		Labels            map[string]string `yaml:"Labels"`
	} `yaml:"StorageClasses"`
}
type NameSpaceRoleVals struct {
	NameSpaceRoleDetails struct {
		AppendName  string            `yaml:"AppendName"`
		Labels      map[string]string `yaml:"Labels"`
		PolicyRules []struct {
			APIGroups []string `yaml:"APIGroups"`
			Resources []string `yaml:"Resources"`
			Verbs     []string `yaml:"Verbs"`
		} `yaml:"PolicyRules"`
	} `yaml:"NameSpaceRoleDetails"`
}
type DefaultQuotaVals struct {
	DefaultQuota struct {
		Details []struct {
			Name                 v1.LimitType                          `yaml:"Name"`
			Max                  map[v1.ResourceName]resource.Quantity `yaml:"max"`
			Min                  map[v1.ResourceName]resource.Quantity `yaml:"min"`
			Default              map[v1.ResourceName]resource.Quantity `yaml:"default,omitempty"`
			DefaultRequest       map[v1.ResourceName]resource.Quantity `yaml:"defaultRequest,omitempty"`
			MaxLimitRequestRatio map[v1.ResourceName]resource.Quantity `yaml:"Details"`
		} `yaml:"Details"`
		Labels map[string]string `yaml:"Labels"`
	} `yaml:"DefaultQuota"`
}

//type InitialConfigVals struct {
//	ClusterDetails struct {
//		ClusterName       string `yaml:"ClusterName"`
//		MasterKey         string `yaml:"Masterkey"`
//		MasterUrl         string `yaml:"Masterurl"`
//		KubeConfig        string `yaml:"kubeconfig"`
//		Configs           string `yaml:"Configs"`
//		StorageClassFile  string  `yaml:"StorageClassesFile"`
//		NameSpaceFile     string `yaml:"NameSpaceFile"`
//		ResourceQuotaFile string `yaml:"ResourceQuotaFile"`
//	} `yaml:"ClusterDetails"`
//}

func SetupConnection(url string, kubeconfig string) (*kubernetes.Clientset, error) {

	config, err := clientcmd.BuildConfigFromFlags(url, kubeconfig)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return c, err
}
func CreateorUpdateStorageClass(config string, connection *kubernetes.Clientset, key string) error {

	con := connection
	var StorageClassVals StorageClassVals

	fileNameSpace, err := ioutil.ReadFile(config)
	if err != nil {
		fmt.Println(err)
		//panic(err)
	}

	err = yaml.Unmarshal([]byte(fileNameSpace), &StorageClassVals)
	if err != nil {
		panic(err)
	}

	var reclaimPolicy v1.PersistentVolumeReclaimPolicy
	var vbmode storagev1.VolumeBindingMode
	var name, Provisioner string
	LenSC := len(StorageClassVals.StorageClasses)

	mapLabels := make(map[string]string)
	mapParams := make(map[string]string)
	m := make(map[string]string)

	m["MasterKey"] = key

	for key, value := range m {
		strKey := fmt.Sprintf("%v", key)
		strValue := fmt.Sprintf("%v", value)
		mapLabels[strKey] = strValue
	}

	ListSC, err := con.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("MasterKey=%s", key)})
	listsc := len(ListSC.Items)
	for i := 0; i < listsc; i++ {
		fmt.Println("StorageClasses Managed by cli: ", ListSC.Items[i].Name)
		//fmt.Println(ListSC.Items[i].Labels)
	}

	if LenSC != 0 && listsc > LenSC {
		if StorageClassVals.Delete.Enable == true {
			fmt.Println("Removing StorageClasses from Clusters")
			for j := 0; j < listsc; j++ {
				for i := 0; i < LenSC; i++ {
					if ListSC.Items[j].Name == StorageClassVals.StorageClasses[i].Name {
						fmt.Println("StorageClass needed: ", StorageClassVals.StorageClasses[i].Name)
					} else {
						fmt.Println("StorageClass not needed, deleting....: ", StorageClassVals.StorageClasses[i].Name)
						DeleteSC := con.StorageV1().StorageClasses().Delete(context.TODO(), ListSC.Items[j].Name, metav1.DeleteOptions{})
						fmt.Println(DeleteSC)
					}
				}
			}
		} else {
			for j := 0; j < listsc; j++ {
				for i := 0; i < LenSC; i++ {
					if ListSC.Items[j].Name == StorageClassVals.StorageClasses[i].Name {
						fmt.Println("StorageClass needed: ", StorageClassVals.StorageClasses[i].Name)
					} else {
						fmt.Println("StorageClass not needed, deleting....: ", StorageClassVals.StorageClasses[i].Name)
						fmt.Println("Removing StorageClasses is not enabled")
					}
				}
			}
		}
	} else if LenSC == 0 && listsc > LenSC {

		if StorageClassVals.Delete.Enable == true {
			fmt.Println("Removing StorageClasses from Clusters")
			for j := 0; j < listsc; j++ {
				fmt.Println("StorageClass not needed, deleting....: ", ListSC.Items[j].Name)
				DeleteSC := con.StorageV1().StorageClasses().Delete(context.TODO(), ListSC.Items[j].Name, metav1.DeleteOptions{})
				fmt.Println(DeleteSC)
			}
		}
	} else {
		for j := 0; j < listsc; j++ {
			fmt.Println("StorageClass not needed, deleting....: ", ListSC.Items[j].Name)
			fmt.Println("Removing StorageClasses is not enabled")
		}

	}
	if LenSC == 0 {
		fmt.Println("No StorageClasses listed to create")
		return nil
	}
	for i := 0; i < LenSC; i++ {

		for key, value := range StorageClassVals.StorageClasses[i].Parameters {
			strKey := fmt.Sprintf("%v", key)
			strValue := fmt.Sprintf("%v", value)
			mapParams[strKey] = strValue
		}

		for key, value := range StorageClassVals.StorageClasses[i].Labels {
			strKey := fmt.Sprintf("%v", key)
			strValue := fmt.Sprintf("%v", value)
			mapLabels[strKey] = strValue
		}

		if StorageClassVals.StorageClasses[i].ReclaimPolicy == "" {
			reclaimPolicy = v1.PersistentVolumeReclaimRetain
		} else if StorageClassVals.StorageClasses[i].ReclaimPolicy == "Retain" {
			reclaimPolicy = v1.PersistentVolumeReclaimRetain
		} else if StorageClassVals.StorageClasses[i].ReclaimPolicy == "Recycle" {
			reclaimPolicy = v1.PersistentVolumeReclaimRecycle
		} else if StorageClassVals.StorageClasses[i].ReclaimPolicy == "Delete" {
			reclaimPolicy = v1.PersistentVolumeReclaimDelete
		} else {
			fmt.Println("Reclaim Policy is not correct")
			return err
		}

		//vbmode := storagev1.VolumeBindingImmediate
		if StorageClassVals.StorageClasses[i].VolumeBindingMode == "" {
			vbmode = storagev1.VolumeBindingWaitForFirstConsumer
		} else if StorageClassVals.StorageClasses[i].VolumeBindingMode == "Immediate" {
			vbmode = storagev1.VolumeBindingImmediate
		} else if StorageClassVals.StorageClasses[i].VolumeBindingMode == "WaitForConsumer" {
			vbmode = storagev1.VolumeBindingWaitForFirstConsumer
		} else {
			fmt.Println("Volume Binding Mode is not correct")
			return err
		}

		if StorageClassVals.StorageClasses[i].Name == "" {
			name = "standard-local"
		} else {
			name = StorageClassVals.StorageClasses[i].Name
		}

		if StorageClassVals.StorageClasses[i].Provisioner == "" {
			Provisioner = "kubernetes.io/no-provisioner"
		} else {
			Provisioner = StorageClassVals.StorageClasses[i].Provisioner
		}

		storageclassjson := storagev1.StorageClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: mapLabels,
			},
			Provisioner:       Provisioner,
			Parameters:        mapParams,
			ReclaimPolicy:     &reclaimPolicy,
			VolumeBindingMode: &vbmode,
		}

		fmt.Println("Storage Class ID: ", i)
		fmt.Println("Storage Class Name: ", name)
		fmt.Println("Storage Class Labels: ", mapLabels)
		fmt.Println("Storage Class Provisioner: ", Provisioner)
		fmt.Println("Storage Class ReclaimPolicy: ", reclaimPolicy)
		fmt.Println("Storage Class VolumeBindingMode: ", vbmode)
		fmt.Println("Storage Class Parameters: ", mapParams)

		CreateSC, err := con.StorageV1().StorageClasses().Create(context.TODO(), &storageclassjson, metav1.CreateOptions{})
		if err != nil {
			fmt.Println(err)
			fmt.Println("Updating StorageClass.....")

			storageclassjsonupdate := storagev1.StorageClass{
				TypeMeta: metav1.TypeMeta{
					Kind:       "StorageClass",
					APIVersion: "storage.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   name,
					Labels: mapLabels,
				},
				Provisioner:       Provisioner,
				ReclaimPolicy:     &reclaimPolicy,
				Parameters:        mapParams,
				VolumeBindingMode: &vbmode,
			}
			UpdateSC, err := con.StorageV1().StorageClasses().Update(context.TODO(), &storageclassjsonupdate, metav1.UpdateOptions{})
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Updated StorageClass : ", UpdateSC.Name)
			}

			//return nil
		} else {
			fmt.Println("Created StorageClass : ", CreateSC.Name)
		}

	}
	return err
}
func CreateorUpdateNameSpace(namespaceyaml string, connection *kubernetes.Clientset, key string) error {

	var NameSpaceVals NameSpaceVals
	con := connection
	mapLabels := make(map[string]string)
	m := make(map[string]string)

	m["MasterKey"] = key

	for key, value := range m {
		strKey := fmt.Sprintf("%v", key)
		strValue := fmt.Sprintf("%v", value)
		mapLabels[strKey] = strValue
	}

	fileNameSpace, err := ioutil.ReadFile(namespaceyaml)
	if err != nil {
		fmt.Println(err)
		//panic(err)
	}
	err = yaml.Unmarshal([]byte(fileNameSpace), &NameSpaceVals)
	if err != nil {
		panic(err)
	}

	lenNs := len(NameSpaceVals.NameSpace)

	ListNS, err := con.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("MasterKey=%s", key)})
	listns := len(ListNS.Items)
	for i := 0; i < listns; i++ {
		fmt.Println("NameSpaces Managed by cli: ", ListNS.Items[i].Name)
		//fmt.Println(ListSC.Items[i].Labels)
	}

	if lenNs != 0 && listns > lenNs {
		if NameSpaceVals.Delete.Enable == true {
			fmt.Println("Removing NameSpaces from Clusters")
			for j := 0; j < listns; j++ {
				for i := 0; i < lenNs; i++ {
					if ListNS.Items[j].Name == NameSpaceVals.NameSpace[i].Name {
						fmt.Println("NameSpaces Class needed: ", NameSpaceVals.NameSpace[i].Name)
					} else {
						fmt.Println("NameSpaces not needed, deleting....: ", NameSpaceVals.NameSpace[i].Name)
						DeleteSC := con.StorageV1().StorageClasses().Delete(context.TODO(), ListNS.Items[j].Name, metav1.DeleteOptions{})
						fmt.Println(DeleteSC)
					}
				}
			}
		} else {
			for j := 0; j < listns; j++ {
				for i := 0; i < lenNs; i++ {
					if ListNS.Items[j].Name == NameSpaceVals.NameSpace[i].Name {
						fmt.Println("NameSpace needed: ", NameSpaceVals.NameSpace[i].Name)
					} else {
						fmt.Println("NameSpace not needed, deleting....: ", NameSpaceVals.NameSpace[i].Name)
						fmt.Println("Removing NameSpaces is not enabled")
					}
				}
			}
		}
	} else if lenNs == 0 && listns > lenNs {
		if NameSpaceVals.Delete.Enable == true {
			fmt.Println("Removing NameSpaces from Clusters")
			for j := 0; j < listns; j++ {
				fmt.Println("NameSpaces not needed, deleting....: ", ListNS.Items[j].Name)
				DeleteSC := con.CoreV1().Namespaces().Delete(context.TODO(), ListNS.Items[j].Name, metav1.DeleteOptions{})
				fmt.Println(DeleteSC)
			}
		}
	} else {
		for j := 0; j < listns; j++ {
			fmt.Println("NameSpaces not needed, deleting....: ", ListNS.Items[j].Name)
			fmt.Println("Removing NameSpaces is not enabled")
		}

	}
	if lenNs == 0 {
		fmt.Println("No NameSpaces listed to create")
		return nil
	}

	//Create or Update NameSpace

	for i := 0; i < lenNs; i++ {

		for key, value := range NameSpaceVals.NameSpace[i].Labels {
			strKey := fmt.Sprintf("%v", key)
			strValue := fmt.Sprintf("%v", value)
			mapLabels[strKey] = strValue
		}

		// 	NS Details
		fmt.Println("NameSpace ID: ", i)
		fmt.Println("NameSpace Name:	", NameSpaceVals.NameSpace[i].Name)
		fmt.Println("NameSpace Resource Quota:	", NameSpaceVals.NameSpace[i].ResourceQuota)
		fmt.Println("NameSpace Default Quota:	", NameSpaceVals.NameSpace[i].DefaultQuota)
		//fmt.Println("NameSpace Labels: 		 ", NameSpaceVals.NameSpace[i].Labels)

		namespacejson := v1.Namespace{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   NameSpaceVals.NameSpace[i].Name,
				Labels: mapLabels,
			},
		}

		fmt.Println("NameSpace Labels: ", mapLabels)

		CreateNameSpace, err := con.CoreV1().Namespaces().Create(context.TODO(), &namespacejson, metav1.CreateOptions{})

		if err != nil {
			fmt.Println(err)
			fmt.Println("Updating NameSpace........")
			UpdateNameSpace, _ := con.CoreV1().Namespaces().Update(context.TODO(), &namespacejson, metav1.UpdateOptions{})
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Updated NameSpace: ", UpdateNameSpace.Name)
			}

		} else {
			println("Created NameSpace : ", CreateNameSpace.Name)
			//Catch the resource details for attaching Resources
		}
	}
	return nil
}
func CreateorUpdateResourceQuota(resourcequotayaml string, namespaceyaml string, connection *kubernetes.Clientset, key string) error {

	var NameSpaceVals NameSpaceVals
	var ResourceQuotaVals ResourceQuotaVals
	con := connection
	var CatchCount int
	var UpdateResourceQuota, CreateResourceQuota *v1.ResourceQuota
	mapLabels := make(map[string]string)
	var StorageClassName string
	var ResourceRequestsCPU, ResourceLimitsCPU, ResourceRequestsMemory, ResourceLimitsMemory, ResourcePods, ResourceRequestsStorage, ResourceRequestsEphemeralStorage, ResourceLimitsEphemeralStorage, StorageClassVolume resource.Quantity

	m := make(map[string]string)
	m["MasterKey"] = key

	for key, value := range m {
		strKey := fmt.Sprintf("%v", key)
		strValue := fmt.Sprintf("%v", value)
		mapLabels[strKey] = strValue
	}

	fileNameSpace, err := ioutil.ReadFile(namespaceyaml)
	if err != nil {
		fmt.Println(err)
		//panic(err)
	}
	err = yaml.Unmarshal([]byte(fileNameSpace), &NameSpaceVals)
	if err != nil {
		panic(err)
	}

	fileResourceQuota, err := ioutil.ReadFile(resourcequotayaml)
	if err != nil {
		fmt.Println(err)
		//panic(err)
	}
	err = yaml.Unmarshal([]byte(fileResourceQuota), &ResourceQuotaVals)
	if err != nil {
		panic(err)
	}

	lenNs := len(NameSpaceVals.NameSpace)
	LenRQ := len(ResourceQuotaVals.ResourceQuota)

	//Create or Update ResourceQuota

	for i := 0; i < lenNs; i++ {

		for key, value := range NameSpaceVals.NameSpace[i].Labels {
			strKey := fmt.Sprintf("%v", key)
			strValue := fmt.Sprintf("%v", value)
			mapLabels[strKey] = strValue
		}

		// NS Details
		fmt.Println("NameSpace Selected - ID: ", i)
		fmt.Println("NameSpace Selected - Name: ", NameSpaceVals.NameSpace[i].Name)
		fmt.Println("NameSpace Selected - Labels: ", mapLabels)

		if LenRQ == 0 {
			fmt.Println("Resource Quotas list is not provided")
		} else {
			// Find the matching Resource Quota
			for k := 0; k < LenRQ; k++ {
				if ResourceQuotaVals.ResourceQuota[k].ResourceQuotaName == NameSpaceVals.NameSpace[i].ResourceQuota {
					CatchCount = k
				}
			}
			for key, value := range ResourceQuotaVals.ResourceQuota[CatchCount].Labels {
				strKey := fmt.Sprintf("%v", key)
				strValue := fmt.Sprintf("%v", value)
				mapLabels[strKey] = strValue
			}

			// Count Storage Classes Defined in Resource Quota

			CountStclass := len(ResourceQuotaVals.ResourceQuota[CatchCount].StorageClasses)

			TotalLen := CountStclass + 8

			arroptkey := make([]v1.ResourceName, int(TotalLen))
			arroptValue := make([]resource.Quantity, int(TotalLen))
			arrayresult3 := make(map[v1.ResourceName]resource.Quantity)
			//lists :=  make(map[v1.ResourceName]resource.Quantity)
			arrkey := [8]v1.ResourceName{v1.ResourceRequestsCPU, v1.ResourceLimitsCPU, v1.ResourceRequestsMemory, v1.ResourceLimitsMemory, v1.ResourcePods, v1.ResourceRequestsStorage, v1.ResourceRequestsEphemeralStorage, v1.ResourceLimitsEphemeralStorage}

			for i := 0; i < 8; i++ {
				arroptkey[i] = arrkey[i]
			}

			if ResourceQuotaVals.ResourceQuota[CatchCount].RequestsCPU == "" {
				ResourceRequestsCPU = resource.MustParse("1")
			} else {
				ResourceRequestsCPU = resource.MustParse(ResourceQuotaVals.ResourceQuota[CatchCount].RequestsCPU)
			}
			if ResourceQuotaVals.ResourceQuota[CatchCount].LimitsCPU == "" {
				ResourceLimitsCPU = resource.MustParse("2")
			} else {
				ResourceLimitsCPU = resource.MustParse(ResourceQuotaVals.ResourceQuota[CatchCount].LimitsCPU)
			}
			if ResourceQuotaVals.ResourceQuota[CatchCount].RequestsMemory == "" {
				ResourceRequestsMemory = resource.MustParse("10Mi")
			} else {
				ResourceRequestsMemory = resource.MustParse(ResourceQuotaVals.ResourceQuota[CatchCount].RequestsMemory)
			}
			if ResourceQuotaVals.ResourceQuota[CatchCount].LimitsMemory == "" {
				ResourceLimitsMemory = resource.MustParse("10Mi")
			} else {
				ResourceLimitsMemory = resource.MustParse(ResourceQuotaVals.ResourceQuota[CatchCount].LimitsMemory)
			}
			if ResourceQuotaVals.ResourceQuota[CatchCount].Pods == "" {
				ResourcePods = resource.MustParse("100")
			} else {
				ResourcePods = resource.MustParse(ResourceQuotaVals.ResourceQuota[CatchCount].Pods)
			}
			if ResourceQuotaVals.ResourceQuota[CatchCount].RequestsStorage == "" {
				ResourceRequestsStorage = resource.MustParse("10M")
			} else {
				ResourceRequestsStorage = resource.MustParse(ResourceQuotaVals.ResourceQuota[CatchCount].RequestsStorage)
			}
			if ResourceQuotaVals.ResourceQuota[CatchCount].RequestsEphemeralStorage == "" {
				ResourceRequestsEphemeralStorage = resource.MustParse("10M")
			} else {
				ResourceRequestsEphemeralStorage = resource.MustParse(ResourceQuotaVals.ResourceQuota[CatchCount].RequestsEphemeralStorage)
			}
			if ResourceQuotaVals.ResourceQuota[CatchCount].LimitsEphemeralStorage == "" {
				ResourceLimitsEphemeralStorage = resource.MustParse("10M")
			} else {
				ResourceLimitsEphemeralStorage = resource.MustParse(ResourceQuotaVals.ResourceQuota[CatchCount].LimitsEphemeralStorage)
			}

			arrVal := [8]resource.Quantity{ResourceRequestsCPU, ResourceLimitsCPU, ResourceRequestsMemory, ResourceLimitsMemory, ResourcePods, ResourceRequestsStorage, ResourceRequestsEphemeralStorage, ResourceLimitsEphemeralStorage}

			for i := 0; i < 8; i++ {
				arroptValue[i] = arrVal[i]
			}

			for i := 0; i < 8; i++ {

				strKey := arroptkey[i]
				strValue := arroptValue[i]
				arrayresult3[strKey] = strValue
			}

			if CountStclass == 0 {

			} else {
				for j := 0; j < CountStclass; j++ {
					if len(ResourceQuotaVals.ResourceQuota[CatchCount].StorageClasses[j].Name) == 0 {
						StorageClassName = "standard-local"
					} else {
						StorageClassName = ResourceQuotaVals.ResourceQuota[CatchCount].StorageClasses[j].Name
					}
					if len(ResourceQuotaVals.ResourceQuota[CatchCount].StorageClasses[j].RequestsStorage) == 0 {
						StorageClassVolume = resource.MustParse("10M")
					} else {
						StorageClassVolume = resource.MustParse(ResourceQuotaVals.ResourceQuota[CatchCount].StorageClasses[j].RequestsStorage)
					}

					fmt.Println("Adding Strorage Class: ", StorageClassName, StorageClassVolume.String())

					arroptkey[8+j] = V1ResourceByStorageClass(StorageClassName, v1.ResourceRequestsStorage)

					arroptValue[8+j] = StorageClassVolume

					strKey := arroptkey[8+j]
					strValue := arroptValue[8+j]
					arrayresult3[strKey] = strValue
				}

			}
			fmt.Println(arrayresult3)

			resourcequotajson := v1.ResourceQuota{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ResourceQuota",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      NameSpaceVals.NameSpace[i].Name + "-resquota",
					Namespace: NameSpaceVals.NameSpace[i].Name,
					Labels:    mapLabels,
				},
				Spec: v1.ResourceQuotaSpec{
					Hard: arrayresult3,
				},
			}

			fmt.Println("Resource ID: ", i)
			fmt.Println("Resource Name: ", NameSpaceVals.NameSpace[i].Name+"-resquota")
			fmt.Println("Resource NameSpace: ", NameSpaceVals.NameSpace[i].Name)
			fmt.Println("Resource Labels: ", mapLabels)
			fmt.Println("Resource Hard Limits: ", arrayresult3)

			CreateResourceQuota, err = con.CoreV1().ResourceQuotas(NameSpaceVals.NameSpace[i].Name).Create(context.TODO(), &resourcequotajson, metav1.CreateOptions{})

			if err != nil {
				fmt.Println(err)
				fmt.Println("Updating ResourceQuota........")
				UpdateResourceQuota, err = con.CoreV1().ResourceQuotas(NameSpaceVals.NameSpace[i].Name).Update(context.TODO(), &resourcequotajson, metav1.UpdateOptions{})
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println("Updated ResourceQuota: ", UpdateResourceQuota.Name)
				}
			} else {
				fmt.Println("Created ResourceQuota: ", CreateResourceQuota.Name)
			}
		}
	}
	return nil
}
func CreateorUpdateDefaultQuota(config string, namespaceyaml string, connection *kubernetes.Clientset, key string) {

	con := connection
	var DefaultQuotaVals DefaultQuotaVals
	var NameSpaceVals NameSpaceVals

	fileNameSpace, err := ioutil.ReadFile(namespaceyaml)
	if err != nil {
		fmt.Println(err)
		//panic(err)
	}
	err = yaml.Unmarshal([]byte(fileNameSpace), &NameSpaceVals)
	if err != nil {
		panic(err)
	}

	fileNameSpace, err = ioutil.ReadFile(config + "/DefaultQuota.yaml")
	if err != nil {
		fmt.Println(err)
		//panic(err)
	}
	err = yaml.Unmarshal([]byte(fileNameSpace), &DefaultQuotaVals)
	if err != nil {
		panic(err)
	}

	LenNS := len(NameSpaceVals.NameSpace)
	LenLR := len(DefaultQuotaVals.DefaultQuota.Details)
	//fmt.Println(LenLR)
	mapLabels := make(map[string]string)

	m := make(map[string]string)
	m["MasterKey"] = key

	for key, value := range m {
		strKey := fmt.Sprintf("%v", key)
		strValue := fmt.Sprintf("%v", value)
		mapLabels[strKey] = strValue
	}

	LimitRangeItem := make([]v1.LimitRangeItem, LenLR)

	for j := 0; j < LenLR; j++ {
		LimitRangeItem[j] = v1.LimitRangeItem{
			Type:                 DefaultQuotaVals.DefaultQuota.Details[j].Name,
			Max:                  DefaultQuotaVals.DefaultQuota.Details[j].Max,
			Min:                  DefaultQuotaVals.DefaultQuota.Details[j].Min,
			Default:              DefaultQuotaVals.DefaultQuota.Details[j].Default,
			DefaultRequest:       DefaultQuotaVals.DefaultQuota.Details[j].DefaultRequest,
			MaxLimitRequestRatio: DefaultQuotaVals.DefaultQuota.Details[j].MaxLimitRequestRatio,
		}
	}

	for key, value := range DefaultQuotaVals.DefaultQuota.Labels {
		strKey := fmt.Sprintf("%v", key)
		strValue := fmt.Sprintf("%v", value)
		mapLabels[strKey] = strValue
	}

	for i := 0; i < LenNS; i++ {

		fmt.Println("NameSpace Selected - ID: ", i)
		fmt.Println("NameSpace Selected - Name: ", NameSpaceVals.NameSpace[i].Name)
		fmt.Println("NameSpace Selected - Labels: ", mapLabels)

		defaultquotajson := v1.LimitRange{
			TypeMeta: metav1.TypeMeta{
				Kind:       "LimitRange",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      NameSpaceVals.NameSpace[i].Name + "-defaultquota",
				Labels:    mapLabels,
				Namespace: NameSpaceVals.NameSpace[i].Name,
			},
			Spec: v1.LimitRangeSpec{
				Limits: LimitRangeItem,
			},
		}

		fmt.Println("DefaultQuota Name ", NameSpaceVals.NameSpace[i].Name+"-defaultquota")
		fmt.Println("DefaultQuota NameSpace: ", NameSpaceVals.NameSpace[i].Name)
		fmt.Println("DefaultQuota Labels: ", mapLabels)
		fmt.Println("DefaultQuota LimitRanges: ", LimitRangeItem)

		CreateDefaultQuota, err := con.CoreV1().LimitRanges(NameSpaceVals.NameSpace[i].Name).Create(context.TODO(), &defaultquotajson, metav1.CreateOptions{})
		if err != nil {
			fmt.Println(err)
			fmt.Println("Updating ServiceAccount.....")
			UpdateSerAcc, _ := con.CoreV1().LimitRanges(NameSpaceVals.NameSpace[i].Name).Update(context.TODO(), &defaultquotajson, metav1.UpdateOptions{})
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Updated ServiceAccount: ", UpdateSerAcc.Name)
			}
		} else {
			fmt.Println("Created ServiceAccount: ", CreateDefaultQuota.Name)
		}
	}
}
func CreateorUpdateNameSpaceUser(config string, namespaceyaml string, connection *kubernetes.Clientset, key string) error {

	var NameSpaceVals NameSpaceVals
	var NameSpaceRoleVals NameSpaceRoleVals
	con := connection
	mapLabels := make(map[string]string)

	m := make(map[string]string)
	m["MasterKey"] = key

	for key, value := range m {
		strKey := fmt.Sprintf("%v", key)
		strValue := fmt.Sprintf("%v", value)
		mapLabels[strKey] = strValue
	}

	fileNameSpace, err := ioutil.ReadFile(namespaceyaml)
	if err != nil {
		fmt.Println(err)
		//panic(err)
	}
	err = yaml.Unmarshal([]byte(fileNameSpace), &NameSpaceVals)
	if err != nil {
		panic(err)
	}

	fileNameSpace, err = ioutil.ReadFile(config + "/DefaultNameSpaceRole.yaml")
	if err != nil {
		fmt.Println(err)
		//panic(err)
	}
	err = yaml.Unmarshal([]byte(fileNameSpace), &NameSpaceRoleVals)
	if err != nil {
		panic(err)
	}

	lenNs := len(NameSpaceVals.NameSpace)
	for key, value := range NameSpaceRoleVals.NameSpaceRoleDetails.Labels {
		strKey := fmt.Sprintf("%v", key)
		strValue := fmt.Sprintf("%v", value)
		mapLabels[strKey] = strValue
	}

	lenPL := len(NameSpaceRoleVals.NameSpaceRoleDetails.PolicyRules)
	PolicyRule := make([]rbacv1.PolicyRule, int(lenPL))
	for j := 0; j < lenPL; j++ {
		PolicyRule[j] = rbacv1.PolicyRule{
			Verbs:     NameSpaceRoleVals.NameSpaceRoleDetails.PolicyRules[j].Verbs,
			APIGroups: NameSpaceRoleVals.NameSpaceRoleDetails.PolicyRules[j].APIGroups,
			Resources: NameSpaceRoleVals.NameSpaceRoleDetails.PolicyRules[j].Resources,
		}
	}

	//fmt.Println(PolicyRule)
	//Create or Update ServiceAccount
	for i := 0; i < lenNs; i++ {

		// NS Details

		fmt.Println("NameSpace Selected - ID: ", i)
		fmt.Println("NameSpace Selected - Name: ", NameSpaceVals.NameSpace[i].Name)
		fmt.Println("NameSpace Selected - Labels: ", mapLabels)

		//create sa
		sajson := v1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceAccount",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      NameSpaceVals.NameSpace[i].Name + "-" + NameSpaceRoleVals.NameSpaceRoleDetails.AppendName + "-sericeaccount",
				Labels:    mapLabels,
				Namespace: NameSpaceVals.NameSpace[i].Name,
			},
		}

		// NS Details
		fmt.Println("ServiceAccount Name: ", NameSpaceVals.NameSpace[i].Name+"-"+NameSpaceRoleVals.NameSpaceRoleDetails.AppendName+"-sericeaccount")
		fmt.Println("ServiceAccount Labels: ", mapLabels)
		fmt.Println("ServiceAccount NameSpace: ", NameSpaceVals.NameSpace[i].Name)

		CreateSerAcc, err := con.CoreV1().ServiceAccounts(NameSpaceVals.NameSpace[i].Name).Create(context.TODO(), &sajson, metav1.CreateOptions{})
		if err != nil {
			fmt.Println(err)
			fmt.Println("Updating ServiceAccount.......")
			UpdateSerAcc, _ := con.CoreV1().ServiceAccounts(NameSpaceVals.NameSpace[i].Name).Update(context.TODO(), &sajson, metav1.UpdateOptions{})
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Updated ServiceAccount: ", UpdateSerAcc.Name)
			}
		} else {
			fmt.Println("Created ServiceAccount: ", CreateSerAcc.Name)
		}

		// Attaching Role

		rolejson := rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Role",
				APIVersion: "rbac.authorization.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      NameSpaceVals.NameSpace[i].Name + "-" + NameSpaceRoleVals.NameSpaceRoleDetails.AppendName + "-role",
				Labels:    mapLabels,
				Namespace: NameSpaceVals.NameSpace[i].Name,
			},
			Rules: PolicyRule,
		}

		fmt.Println("Role Name: ", NameSpaceVals.NameSpace[i].Name+"-"+NameSpaceRoleVals.NameSpaceRoleDetails.AppendName+"-role")
		fmt.Println("Role Labels: ", mapLabels)
		fmt.Println("Role NameSpace: ", NameSpaceVals.NameSpace[i].Name)
		fmt.Println("Role PolicyRule: ", PolicyRule)

		CreateRole, err := con.RbacV1().Roles(NameSpaceVals.NameSpace[i].Name).Create(context.TODO(), &rolejson, metav1.CreateOptions{})
		if err != nil {
			fmt.Println(err)
			fmt.Println("Updating Role........")
			UpateRole, _ := con.RbacV1().Roles(NameSpaceVals.NameSpace[i].Name).Update(context.TODO(), &rolejson, metav1.UpdateOptions{})
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("Updated Role: ", UpateRole.Name)
		} else {
			fmt.Println("Created Role: ", CreateRole.Name)
		}

		rolebdjson := rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleBinding",
				APIVersion: "rbac.authorization.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      NameSpaceVals.NameSpace[i].Name + "-" + NameSpaceRoleVals.NameSpaceRoleDetails.AppendName + "-rolebinding",
				Labels:    mapLabels,
				Namespace: NameSpaceVals.NameSpace[i].Name,
			},
			Subjects: []rbacv1.Subject{
				rbacv1.Subject{
					Kind:      "ServiceAccount",
					Namespace: NameSpaceVals.NameSpace[i].Name,
					Name:      NameSpaceVals.NameSpace[i].Name + "-" + NameSpaceRoleVals.NameSpaceRoleDetails.AppendName + "-sericeaccount",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     NameSpaceVals.NameSpace[i].Name + "-" + NameSpaceRoleVals.NameSpaceRoleDetails.AppendName + "-sericeaccount",
			},
		}

		fmt.Println("RoleBinding Name: ", NameSpaceVals.NameSpace[i].Name+"-"+NameSpaceRoleVals.NameSpaceRoleDetails.AppendName+"-rolebinding")
		fmt.Println("RoleBinding Labels: ", mapLabels)
		fmt.Println("RoleBinding NameSpace ", NameSpaceVals.NameSpace[i].Name)
		fmt.Println("RoleBinding ServiceAccount: ", NameSpaceVals.NameSpace[i].Name+"-"+NameSpaceRoleVals.NameSpaceRoleDetails.AppendName+"-sericeaccount")
		fmt.Println("RoleBinding Role: ", NameSpaceVals.NameSpace[i].Name+"-"+NameSpaceRoleVals.NameSpaceRoleDetails.AppendName+"-sericeaccount")

		CreateRoleBinding, err := con.RbacV1().RoleBindings(NameSpaceVals.NameSpace[i].Name).Create(context.TODO(), &rolebdjson, metav1.CreateOptions{})
		if err != nil {
			fmt.Println(err)
			fmt.Println("Updating RoleBinding........")
			UpdateRoleBinding, _ := con.RbacV1().RoleBindings(NameSpaceVals.NameSpace[i].Name).Update(context.TODO(), &rolebdjson, metav1.UpdateOptions{})
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Updated RoleBinding: ", UpdateRoleBinding.Name)
			}
			//return nil
		} else {
			fmt.Println("Created RoleBinding: ", CreateRoleBinding.Name)
		}
	}

	return err
}
func V1ResourceByStorageClass(storageClass string, resourceName v1.ResourceName) v1.ResourceName {
	return v1.ResourceName(string(storageClass + storageClassSuffix + string(resourceName)))
}
func Init(clustername string, masterurl string, kubeconfig string) (err error) {

	// Variables - host, namespace
	filePath := "K8Cli" + "/mgmt/" + clustername
	mgmtpath := "K8Cli/mgmt/" + clustername
	configpath := "K8Cli/mgmt/" + clustername + "/configs"
	storageclasspath := "K8Cli/mgmt/" + clustername + "/StorageClasses"
	namespacepath := "K8Cli/mgmt/" + clustername + "/NameSpaces"
	resourcequotapath := "K8Cli/mgmt/" + clustername + "/ResourceQuotas"

	storageclassfile := storageclasspath + "/StorageClasses.yaml"
	namespacefile := namespacepath + "/Namespaces.yaml"
	resourcequotafile := resourcequotapath + "/ResourceQuota.yaml"

	//println(data)
	_, err = os.Stat(filePath)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(filePath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		b := make([]byte, 4)
		rand.Read(b)
		token := fmt.Sprintf("%x", b)
		fmt.Println("Token generated: ", token)
		//type AutoGenerated struct {

		type ClusterDetails struct {
			ClusterName       string `yaml:"ClusterName"`
			MasterKey         string `yaml:"Masterkey"`
			MasterUrl         string `yaml:"Masterurl"`
			KubeConfig        string `yaml:"Kubeconfig"`
			Configs           string `yaml:"Configs"`
			StorageClassFile  string `yaml:"StorageClassfile"`
			NameSpaceFile     string `yaml:"NameSpacefile"`
			ResourceQuotaFile string `yaml:"ResourceQuotafile"`
		}
		var data = `
---
ClusterDetails:
  ClusterName: {{ .ClusterName }}
  MasterKey: {{ .MasterKey }}
  MasterUrl: {{ .MasterUrl }}
  kubeConfig: {{ .KubeConfig }}
  Configs: {{ .Configs }}
  StorageClassFile: {{ .StorageClassFile }}
  NameSpaceFile: {{ .NameSpaceFile }}
  ResourceQuotaFile: {{ .ResourceQuotaFile }}
`

		// Create the file:
		err = ioutil.WriteFile(filePath+"/config.tmpl", []byte(data), 0644)
		check(err)

		values := ClusterDetails{ClusterName: clustername, MasterKey: token, MasterUrl: masterurl, KubeConfig: kubeconfig, Configs: configpath, StorageClassFile: storageclassfile, NameSpaceFile: namespacefile, ResourceQuotaFile: resourcequotafile}

		//values := ClusterDetails{ClusterName: "", Masterkey: "", Masterurl: "", Kubeconfig: "", Config: "", Namespacefile: "", Resourcequotafile: ""}

		var templates *template.Template
		var allFiles []string

		if err != nil {
			fmt.Println(err)
		}

		//for _, file := range files {
		filename := "config.tmpl"
		fullPath := filePath + "/config.tmpl"
		if strings.HasSuffix(filename, ".tmpl") {
			allFiles = append(allFiles, fullPath)
		}
		//}
		fmt.Println(allFiles)
		templates, err = template.ParseFiles(allFiles...)
		if err != nil {
			fmt.Println(err)
		}

		s1 := templates.Lookup("config.tmpl")
		//f, err := os.Create(filePath+"/config.yaml")
		f, err := os.Create(filePath + "/config.yaml")
		if err != nil {
			panic(err)
		}
		//defer f.Close() // don't forget to close the file when finished.

		fmt.Println("Creating .K8Cli folder and config files")
		// Write template to file:
		err = s1.Execute(f, values)
		defer f.Close() // don't forget to close the file when finished.
		if err != nil {
			panic(err)
		}
	} else {

		fmt.Println(".K8Cli/mgmt/<cluster> exists, please manually edit file to make changes or provide new cluster name")

	}

	_, err = os.Stat(mgmtpath)
	if os.IsNotExist(err) {

		fmt.Println("Creating K8Cli/mgmt/<cluster> folder")
		errDir := os.MkdirAll(mgmtpath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("K8Cli/mgmt/<cluster> exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(configpath)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(configpath, 0755)

		var DefaultQuota = `
---
DefaultQuota:
  Details:
    - Name: Container
      max:
        cpu: 2
        memory: 1Gi
      min:
        cpu: 100m
        memory: 4Mi
      default:
        cpu: 300m
        memory: 200Mi
      defaultRequest:
        cpu: 200m
        memory: 100Mi
      maxLimitRequestRatio:
        cpu: 10
    - Name: Pod
      max:
        cpu: 2
        memory: 1Gi
      min:
        cpu: 200m
        memory: 6Mi
  Labels:
   Key1: Val1
   Key2: Val2
`
		var DefaultRole = `
---
NameSpaceRoleDetails:
  AppendName: test123
  PolicyRules:
    - APIGroups: ["","extensions","apps"]
      Resources: ["*"]
      Verbs:     ["*"]
    - APIGroups: ["batch"]
      Resources: ["jobs","cronjobs"]
      Verbs:     ["*"]
  Labels:
    Key1: Val1
`
		fmt.Println("Creating K8Cli/mgmt/<cluster>/configs and sample files")
		err = ioutil.WriteFile(configpath+"/DefaultQuota.yaml", []byte(DefaultQuota), 0644)
		check(err)
		err = ioutil.WriteFile(configpath+"/DefaultNameSpaceRole.yaml", []byte(DefaultRole), 0644)
		check(err)

		if errDir != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("K8Cli/mgmt/<cluster>/configs exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(storageclasspath)
	if os.IsNotExist(err) {

		errDir := os.MkdirAll(storageclasspath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		var SampleStorageYaml = `
---
Delete:
  Enable: True
StorageClasses:
  - Name:
    Provisioner:
    Parameters:
      Key1 : Val1
    ReclaimPolicy:
    VolumeBindingMode:
    Labels:
      Key1: Val1
      Key2: Val2
  - Name: slow
    Provisioner: kubernetes.io/azure-disk
    Parameters:
      skuName: Standard_LRS
      location: eastus
      storageAccount: azure_storage_account_test
    ReclaimPolicy:
    VolumeBindingMode:
    Labels:
      Key1: Val1
      Key2: Val2
`
		fmt.Println("Creating K8Cli/mgmt/<cluster>/StorageClasses folder and sample files")
		err = ioutil.WriteFile(storageclassfile, []byte(SampleStorageYaml), 0644)
		check(err)

	} else {
		fmt.Println("K8Cli/mgmt/<cluster>/StorageClasses exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(namespacepath)
	if os.IsNotExist(err) {

		errDir := os.MkdirAll(namespacepath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		var SampleNameSpaceYaml = `
---
Delete:
  Enable: True
NameSpace:
  - Name: "test"
    ResourceQuota: "q1"
    DefaultQuota: " "
    Labels:
      "Key1": "Val1"
  - Name: "test1"
    ResourceQuota: "q1"
    DefaultQuota: " "
    Labels:
      "Key1": "Val1"
`

		fmt.Println("Creating K8Cli/mgmt/<cluster>/NameSpaces folder and sample files")
		err = ioutil.WriteFile(namespacefile, []byte(SampleNameSpaceYaml), 0644)
		check(err)

	} else {
		fmt.Println("K8Cli/mgmt/<cluster>/NameSpaces exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(resourcequotapath)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(resourcequotapath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		var SampleResourceYaml = `
---
ResourceQuota:
  - QuotaName: "q1"
    RequestsCPU: 10
    LimitsCPU: 10
    RequestsMemory: 10
    LimitsMemory: 10
    Pods: 40
    RequestsStorage: 10
    RequestsEphemeralStorage: 10
    LimitsEphemeralStorage: 10
    StorageClasses:
      - Name:
        RequestsStorage: 5G
      - Name:
        RequestsStorage: 20G
      - Name:
        RequestsStorage: 40G
    Labels:
      "Key1": "Val1"
      "Key2": "Val2"
`
		fmt.Println("Creating K8Cli/mgmt/<cluster>/ResourceQuotas folder and sample files")
		err = ioutil.WriteFile(resourcequotafile, []byte(SampleResourceYaml), 0644)
		check(err)

	} else {
		fmt.Println("K8Cli/mgmt/<cluster>/ResourceQuotas exists, please manually edit file to make changes or provide new cluster name")
	}

	return
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}
