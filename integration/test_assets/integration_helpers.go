package test_assets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"k8s.io/client-go/pkg/api/v1"
	v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"

	. "github.com/onsi/gomega"
)

type cpiTemplate struct {
	Context  string
	DiskID   string
	AgentID  string
	Replicas string
}

type KubeConfigTemplate struct {
	ClusterName  string
	Cert         string
	User         string
	APIServer    string
	RefreshToken string
	Token        string
}

type KubeConfig struct {
	Kind        string `yaml:"kind"`
	APIVersion  string `yaml:"apiVersion"`
	Preferences struct {
	} `yaml:"preferences"`
	Clusters []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			Server               string `yaml:"server"`
			CertificateAuthority string `yaml:"certificate-authority"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Users []struct {
		Name string `yaml:"name"`
		User struct {
			AuthProvider struct {
				Name   string `yaml:"name"`
				Config struct {
					ClientID     string `yaml:"client-id"`
					ClientSecret string `yaml:"client-secret"`
					IDToken      string `yaml:"id-token"`
					IdpIssuerURL string `yaml:"idp-issuer-url"`
					RefreshToken string `yaml:"refresh-token"`
				} `yaml:"config"`
			} `yaml:"auth-provider"`
		} `yaml:"user"`
	} `yaml:"users"`
	Contexts []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster   string `yaml:"cluster"`
			User      string `yaml:"user"`
			Namespace string `yaml:"namespace"`
		} `yaml:"context"`
	} `yaml:"contexts"`
	CurrentContext string `yaml:"current-context"`
}

const templatePath = "../test_assets/cpi_methods"

func ConnectCluster() error {
	bxAPI := os.Getenv("BX_API")
	if bxAPI == "" {
		return errors.New("BX_API must be set")
	}

	bxUsername := os.Getenv("BX_USERNAME")
	if bxUsername == "" {
		return errors.New("BX_USERNAME must be set")
	}

	// optional api key -- only applies to those with federated IDs
	bxAPIKey := os.Getenv("BX_API_KEY")

	// if the api key is provided, no password is required; however, one or the other must be set
	bxPassword := os.Getenv("BX_PASSWORD")
	if bxAPIKey == "" && bxPassword == "" {
		return errors.New("BX_API_KEY or BX_PASSWORD must be set")
	}

	// if the api key is provided, no acct id is required; however, one or the other must be set
	bxAccountID := os.Getenv("BX_ACCOUNTID")
	if bxAPIKey == "" && bxAccountID == "" {
		return errors.New("BX_API_KEY or BX_ACCOUNTID must be set")
	}

	clusterName := os.Getenv("CLUSTER_NAME")
	if clusterName == "" {
		return errors.New("CLUSTER_NAME must be set")
	}

	slUsername := os.Getenv("SL_USERNAME")
	if slUsername == "" {
		return errors.New("SL_USERNAME must be set")
	}

	slAPIKey := os.Getenv("SL_API_KEY")
	if slAPIKey == "" {
		return errors.New("SL_API_KEY must be set")
	}

	loginArgs := []string{"login", "-a", bxAPI}

	// Log in to the Bluemix CLI.
	if bxAPIKey != "" {
		loginArgs = append(loginArgs, "--apikey", bxAPIKey)
	} else {
		loginArgs = append(loginArgs, "-u", bxUsername, "-p", bxPassword, "-c", bxAccountID)
	}

	loginBX := exec.Command("bx", loginArgs...)

	err := loginBX.Run()
	if err != nil {
		return errors.Wrap(err, "Logging in to Bluemix CLI")
	}

	//Initialize the IBM Bluemix Container Service plug-in
	initCS := exec.Command("bx", "cs", "init")
	err = initCS.Run()
	if err != nil {
		return errors.Wrap(err, "Initializing the IBM Bluemix Container Service plug-in")
	}

	//Set Softlayer Credentials
	setSLCredentials := exec.Command("bx", "cs", "credentials-set", "--infrastructure-username", slUsername, "--infrastructure-api-key", slAPIKey)
	err = setSLCredentials.Run()
	if err != nil {
		return errors.Wrap(err, "Setting Softlayer Credentials")
	}

	//Verify provided cluster
	listClusters := exec.Command("bx", "cs", "clusters")
	listClustersOutput, err := listClusters.Output()
	if err != nil {
		return errors.Wrap(err, "Listing Clusters")
	}

	if !strings.Contains(string(listClustersOutput), clusterName) {
		return errors.New(fmt.Sprintf("Cannot find cluster %s", clusterName))
	}

	// TODO: What's the right way to solve this?  Do we revert to downloading the config,
	// or do we specify that the user must set his/her KUBECONFIG before running the tests?

	//Set your terminal context to your cluster
	// setContext := exec.Command("bx", "cs", "cluster-config", clusterName)
	// setContextOutput, err := setContext.Output()
	// if err != nil {
	// 	return errors.Wrap(err, "Setting your terminal context to your cluster")
	// }

	//Export environment variables to start using Kubernetes.
	// kuberConfig := strings.SplitAfter(string(setContextOutput), "KUBECONFIG=")[1]
	kuberConfig := os.Getenv("KUBECONFIG")
	if kuberConfig == "" {
		return errors.New("You must set the KUBECONFIG environment variable before running the tests")
	}

	// env := strings.Replace(kuberConfig, "\n", "", -1)
	// err = os.Setenv("KUBECONFIG", env)
	// if err != nil {
	// 	return errors.Wrap(err, "Exporting  environment variables to start using Kubernetes")
	// }

	return nil
}

func RunCpi(rootCpiPath string, configPath string, agentPath string, jsonPayload string) ([]byte, error) {
	cpiPath := filepath.Join(rootCpiPath, "out/cpi")
	agentAbsPath := filepath.Join(rootCpiPath, agentPath)

	cmd := exec.Command(cpiPath, "--kubeConfig", configPath, "-agentConfig", agentAbsPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return []byte{}, err
	}

	_, err = stdin.Write([]byte(jsonPayload))
	if err != nil {
		return []byte{}, err
	}

	err = stdin.Close()
	if err != nil {
		return []byte{}, err
	}

	var errbuf bytes.Buffer
	cmd.Stderr = &errbuf
	output, err := cmd.Output()
	if err != nil {
		return []byte{}, err
	}

	return output, nil
}

func GenerateCpiJsonPayload(methodName string, rootTemplatePath string, replacementMap map[string]string) (string, error) {
	var (
		val    string
		exists bool
	)

	c := cpiTemplate{
		Context: replacementMap["context"],
	}

	if val, exists = replacementMap["diskID"]; exists {
		c.DiskID = val
	}
	if val, exists = replacementMap["agentID"]; exists {
		c.AgentID = val
	}
	if val, exists = replacementMap["replicas"]; exists {
		c.Replicas = val
	}

	t := template.New(fmt.Sprintf("%s.json", methodName))

	methodPath := filepath.Join(rootTemplatePath, "integration/test_assets/cpi_methods", fmt.Sprintf("%s.json", methodName))
	t, err := t.ParseFiles(methodPath)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = t.Execute(buf, c)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// CreateTmpConfigFile Creates a temporary config file, writes the config to it, and returns the path
func CreateTmpConfigFile(rootTemplatePath string, configPath string, kubeConfig string) (string, error) {
	currentKubeConfig := KubeConfig{}
	configBytes, err := ioutil.ReadFile(kubeConfig)
	if err != nil {
		return "", err
	}
	yaml.Unmarshal(configBytes, &currentKubeConfig)

	var apiServer string
	var certName string
	var refreshToken string
	var token string
	clusterName := currentKubeConfig.CurrentContext

	for _, value := range currentKubeConfig.Clusters {
		apiServer = value.Cluster.Server
		certName = value.Cluster.CertificateAuthority
	}

	for _, value := range currentKubeConfig.Users {
		refreshToken = value.User.AuthProvider.Config.RefreshToken
		token = value.User.AuthProvider.Config.IDToken
	}

	certPath := fmt.Sprintf("%s/.bluemix/plugins/container-service/clusters/%s/%s", os.Getenv("HOME"), clusterName, certName)
	certBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		return "", err
	}

	cert := strconv.Quote(string(certBytes))

	KubeConfigTemplate := KubeConfigTemplate{
		ClusterName:  clusterName,
		Cert:         cert,
		APIServer:    apiServer,
		RefreshToken: refreshToken,
		Token:        token,
	}

	t := template.New("config.json")

	t, err = t.ParseFiles(filepath.Join(rootTemplatePath, configPath))
	if err != nil {
		return "", err
	}

	tempFile, err := ioutil.TempFile("", "config")
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = t.Execute(buf, KubeConfigTemplate)
	if err != nil {
		return "", err
	}

	_, err = tempFile.Write(buf.Bytes())
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func DeleteNamespace(namespace string) {
	deleteNs := exec.Command("kubectl", "delete", "ns", namespace)
	err := deleteNs.Run()
	Expect(err).NotTo(HaveOccurred())
}

func CreateNamespace(namespace string) {
	createNs := exec.Command("kubectl", "create", "ns", namespace)
	err := createNs.Run()
	Expect(err).NotTo(HaveOccurred())
}

func PodCount(namespace string) (int, error) {
	var pods v1.PodList

	cmd := exec.Command("kubectl", "-n", namespace, "get", "po", "-o", "json")
	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}

	if err := json.NewDecoder(cmdOut).Decode(&pods); err != nil {
		return 0, err
	}

	if err := cmd.Wait(); err != nil {
		return 0, errors.New("Failure in Wait() when executing external command")
	}

	// fmt.Fprintln(os.Stderr, "PODS FOUND")
	// for i := 0; i < len(pods.Items); i++ {
	// 	fmt.Fprintf(os.Stderr, "Pod: %s\tNamespace: %s\n", pods.Items[i].ObjectMeta.Name, pods.Items[i].ObjectMeta.Namespace)
	// }
	return len(pods.Items), nil
}

func ReplicaCount(namespace, vmcid string) (int32, error) {
	var deployments v1beta1.DeploymentList

	cmd := exec.Command("kubectl", "-n", namespace, "get", "deployments", "-o", "json")
	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}

	if err := json.NewDecoder(cmdOut).Decode(&deployments); err != nil {
		return 0, err
	}

	if err := cmd.Wait(); err != nil {
		return 0, errors.New("Failure in Wait() when executing external command")
	}

	// Find appropriate deployment and return number of replicas
	for _, deployment := range deployments.Items {
		if deployment.GetObjectMeta().GetName() == "agent-"+vmcid {
			return *deployment.Spec.Replicas, nil
		}
	}

	return 0, fmt.Errorf("could not find deployment agent-%s", vmcid)
}

func ServiceCount(namespace string) (int, error) {
	var services v1.ServiceList

	cmd := exec.Command("kubectl", "-n", namespace, "get", "svc", "-o", "json")
	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}

	if err := json.NewDecoder(cmdOut).Decode(&services); err != nil {
		return 0, err
	}

	if err := cmd.Wait(); err != nil {
		return 0, errors.New("Failure in Wait() when executing external command")
	}

	return len(services.Items), nil
}

func GetServiceByName(namespace string, serviceName string) (v1.Service, error) {
	var service = v1.Service{}

	cmd := exec.Command("kubectl", "-n", namespace, "get", "svc", serviceName, "-o", "json")
	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		return service, err
	}
	if err := cmd.Start(); err != nil {
		return service, err
	}

	if err := json.NewDecoder(cmdOut).Decode(&service); err != nil {
		return service, err
	}

	if err := cmd.Wait(); err != nil {
		return service, errors.New("Failure in Wait() when executing external command")
	}

	return service, nil
}

func Pvcs(namespace string) (v1.PersistentVolumeClaimList, error) {
	var pvcs v1.PersistentVolumeClaimList

	cmd := exec.Command("kubectl", "-n", namespace, "get", "pvc", "-o", "json")
	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		return v1.PersistentVolumeClaimList{}, err
	}
	if err := cmd.Start(); err != nil {
		return v1.PersistentVolumeClaimList{}, err
	}

	if err := json.NewDecoder(cmdOut).Decode(&pvcs); err != nil {
		return v1.PersistentVolumeClaimList{}, err
	}

	if err := cmd.Wait(); err != nil {
		return v1.PersistentVolumeClaimList{}, errors.New("Failure in Wait() when executing external command")
	}

	return pvcs, nil
}

func GetPodByName(podName string, namespace string) (v1.Pod, error) {
	pod := v1.Pod{}

	cmd := exec.Command("kubectl", "-n", namespace, "get", "pod", podName, "-o", "json")
	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		return pod, err
	}
	if err := cmd.Start(); err != nil {
		return pod, err
	}

	if err := json.NewDecoder(cmdOut).Decode(&pod); err != nil {
		return pod, err
	}

	if err := cmd.Wait(); err != nil {
		return pod, errors.New("Failure in Wait() when executing external command")
	}

	return pod, nil
}

func GetPodListByAgentId(namespace string, agentId string) (v1.PodList, error) {
	var pods v1.PodList

	cmd := exec.Command("kubectl", "-n", namespace, "get", "pods", "-l", fmt.Sprintf("bosh.cloudfoundry.org/agent-id=%s", agentId), "-o", "json")
	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		return v1.PodList{}, err
	}
	if err := cmd.Start(); err != nil {
		return v1.PodList{}, err
	}

	if err := json.NewDecoder(cmdOut).Decode(&pods); err != nil {
		return v1.PodList{}, err
	}

	if err := cmd.Wait(); err != nil {
		return v1.PodList{}, errors.New("Failure in Wait() when executing external command")
	}

	return pods, nil
}
