package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v5"
	certmanagerv1 "github.com/cert-manager/cert-manager"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/mittwald/go-helm-client/values"
	"helm.sh/helm/v3/pkg/repo"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	clientcmd "k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v5"
)

func main() {
	// First add the ArgoCD Chart to a repo in the cluster

	var outputBuffer bytes.Buffer
	ctx := context.Background()
	mycontext, _ := context.WithTimeout(ctx, 80000*time.Second)
	///// FLAGS To COME IN Here////
	certmanageremail := "jason.content@gmail.com"
	subid := "AzureSumID"
	rg := "JCKubernetes-newprodjcvs2"
	cluster := "JCKubernetes-newprodjcvs2"

	//////////////////////////
	fmt.Println("Connecting to Cluster Via Azure Call\n")
	myaksconnect := connectToAks(ctx, subid, rg, cluster)
	mypublicip := "0.0.0.0"
	fmt.Printf("Moved %v to KubeConfig File", &myaksconnect.Name) // This is from here: https://github.com/Azure/azure-sdk-for-go/blob/sdk/resourcemanager/containerservice/armcontainerservice/v5.0.0/sdk/resourcemanager/containerservice/armcontainerservice/responses.go#L99
	fmt.Println("Connecting to Cluster using KubeConfig File\n")
	kubeClient := connectToK8s()

	// Builds Helm Chart Client ////
	fmt.Println("Building Helm Client\n")
	opt := &helmclient.Options{
		Namespace:        "default", // Change this to the namespace you wish the client to operate in.
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            false,
		Linting:          true,
		DebugLog:         func(format string, v ...interface{}) {},
		Output:           &outputBuffer, // Not mandatory, leave open for default os.Stdout
	}

	myHelmClient, err := helmclient.New(opt)
	if err != nil {
		panic(err)
	}

	/////////////////////////////////////////////////////////////////////////////////////////////////////
	//// Now use Helmchart client to create external-secrets chart repo //////
	fmt.Println("Deploying nginx-ingress\n")
	nginxchartRepo := repo.Entry{
		Name:               "nginx-ingress",
		URL:                "https://kubernetes.github.io/ingress-nginx",
		PassCredentialsAll: true,
	}

	if err := myHelmClient.AddOrUpdateChartRepo(nginxchartRepo); err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Added Chart Repo %s\n", nginxchartRepo.Name)
	}

	// Now Run Update Chart Repos
	if err := myHelmClient.UpdateChartRepos(); err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Updating Chart Repo\n")
	}

	///////////////////////////////////////////////////////////////////////////////////////////////////////
	// Now install the chart from the repo thats just been created./////////////////////
	nginxchartSpec := helmclient.ChartSpec{
		ReleaseName:     "nginx-ingress/nginx-ingress",
		ChartName:       "nginx-ingress",
		Namespace:       "nginx-ingress",
		CreateNamespace: true,
		SkipCRDs:        false,
		Wait:            true,
		ValuesOptions: values.Options{
			StringValues: []string{"rbac.create=false",
				"controller.service.externalTrafficPolicy=Local",
				fmt.Sprintf("controller.service.loadBalancerIP=%v", mypublicip),
				"controller.replicaCount=2",
				fmt.Sprintf("controller.nodeSelector.kubernetes\\.io/os=linux"),
				"defaultBackend.nodeSelector.kubernetes\\.io/os=linux",
				"controller.admissionWebhooks.patch.nodeSelector.kubernetes\\.io/os=linux",
				"controller.publishService.enabled=true",
				"controller.service.beta.kubernetes.io/azure-load-balancer-health-probe-request-path=/healthz",
			},
		},
	}
	nginxInstalledHelmChart, err := myHelmClient.InstallChart(mycontext, &nginxchartSpec, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Status of Chart Install %v,\n", *nginxInstalledHelmChart.Info)
	//// Now use Helmchart client to create external-secrets chart repo //////
	fmt.Println("Deploying cert-manager\n")
	certmanagerchartRepo := repo.Entry{
		Name:               "cert-manager",
		URL:                "https://charts.jetstack.io",
		PassCredentialsAll: true,
	}

	if err := myHelmClient.AddOrUpdateChartRepo(certmanagerchartRepo); err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Added Chart Repo %s\n", certmanagerchartRepo.Name)
	}

	// Now Run Update Chart Repos
	if err := myHelmClient.UpdateChartRepos(); err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Updating Chart Repo\n")
	}

	///////////////////////////////////////////////////////////////////////////////////////////////////////
	// Now install the chart from the repo thats just been created./////////////////////
	certmanagerchartSpec := helmclient.ChartSpec{
		ReleaseName:     "cert-manager/cert-manager",
		ChartName:       "cert-manager",
		Version:         "v1.14.5",
		Namespace:       "cert-manager",
		CreateNamespace: true,
		ValuesOptions: values.Options{
			StringValues: []string{
				"extraArgs = {--dns01-recursive-nameservers=1.1.1.1:53}",
				"controller.nodeSelector.kubernetes\\.io/os=linux",
			},
		},
		SkipCRDs: false,
		Wait:     true,
	}
	certmanagermyInstalledHelmChart, err := myHelmClient.InstallChart(mycontext, &certmanagerchartSpec, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Status of Chart Install %v,\n", *certmanagermyInstalledHelmChart.Info)

	//// Now use Helmchart client to create external-secrets chart repo //////
	externalsecchartRepo := repo.Entry{
		Name:               "external-secrets",
		URL:                "https://charts.external-secrets.io",
		PassCredentialsAll: true,
	}

	if err := myHelmClient.AddOrUpdateChartRepo(externalsecchartRepo); err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Added Chart Repo %s\n", externalsecchartRepo.Name)
	}

	// Now Run Update Chart Repos
	if err := myHelmClient.UpdateChartRepos(); err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Updating Chart Repo\n")
	}

	///////////////////////////////////////////////////////////////////////////////////////////////////////
	// Deploy Lets Encrypt
	fmt.Println("Deploying Lets Encrypt\n")

	letsencsecret := map[string]string{
		"api-token": "SECRET",
	}

	myLetsEncryptApiKey := core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "letsencrypt-cloudflare-api-token-secret",
		},
		StringData: letsencsecret,
		Type:       "Generic",
	}

	createSecret, err := kubeClient.CoreV1().Secrets("cert-manager").Create(ctx, &myLetsEncryptApiKey, metav1.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Created Secret %v", createSecret.Name)
	}

	certmanagerissuer := certmanagerv1.ClusterIsser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "letsencrypt-jc",
			Namespace: "cert-manager",
		},
		Spec: certmanagerv1.IssuerConfig{
			ACME: certmanagerv1.ACMEIssuer{
				Email:  certmanageremail,
				Server: "https://acme-v02.api.letsencrypt.org/directory",
				PrivateKey: certmanagerv1.SecretKeySelector{
					certmanagerv1.LocalObjectReference{
						Name: "letsencrypt-JC",
					},
					Solvers: certmanagerv1.ACMEChallengeSolverDNS01{
						Cloudflare: certmanagerv1.ACMEIssuerDNS01ProviderCloudflare{
							APIToken: certmanagerv1.SecretKeySelector{
								certmanagerv1.LocalObjectReference{
									Name: "letsencrypt-cloudflare-api-token-secret",
								},
								Key: string(letsencsecret["api-token"]),
							},
						},
						Selector: certmanagerv1.CertificateDNSNameSelector{
							DNSZones: []string{"theclouddude.co.uk"},
						},
					},
				},
			},
		},
	}

	createcertissuer, err := kubeClient.CoreV1().Namespaces().Create(ctx, &core.Namespace{}, &certmanagerissuer)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Status of Apply %v", createcertissuer.Status)

	// Now install the chart from the repo thats just been created./////////////////////
	fmt.Println("Deploying external-secrets\n")
	externalsecchartSpec := helmclient.ChartSpec{
		ReleaseName:     "external-secrets/external-secrets",
		ChartName:       "external-secrets",
		Version:         "0.9.18",
		Namespace:       "external-secrets",
		CreateNamespace: true,
		SkipCRDs:        false,
		Wait:            true,
	}
	externalsexmyInstalledHelmChart, err := myHelmClient.InstallChart(mycontext, &externalsecchartSpec, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Status of Chart Install %v,\n", *externalsexmyInstalledHelmChart.Info)

	////////////////////////////////////////////////////////////////////////////////////////

	//// Now use Helmchart client to create Argo CD chart repo //////
	fmt.Println("Deploying Argo-CD\n")
	ArgoCDchartRepo := repo.Entry{
		Name:               "argo",
		URL:                "https://argoproj.github.io/argo-helm",
		PassCredentialsAll: true,
	}

	if err := myHelmClient.AddOrUpdateChartRepo(ArgoCDchartRepo); err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Added Chart Repo %s\n", ArgoCDchartRepo.Name)
	}

	// Now Run Update Chart Repos
	if err := myHelmClient.UpdateChartRepos(); err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Updating Chart Repo\n")
	}

	///////////////////////////////////////////////////////////////////////////////////////////////////////
	// Now install the chart from the repo thats just been created./////////////////////
	ArgoCDchartSpec := helmclient.ChartSpec{
		ReleaseName:     "argo-cd-chart",
		ChartName:       "argo/argo-cd",
		Version:         "6.9.2",
		Namespace:       "argo-cd",
		CreateNamespace: true,
		ValuesOptions: values.Options{
			Values: []string{"./argocd-values.yaml", "./deployenvcongifmap"},
		},
		SkipCRDs: true,
		Wait:     true,
	}
	argoCDmyInstalledHelmChart, err := myHelmClient.InstallChart(mycontext, &ArgoCDchartSpec, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Status of Chart Install %v,\n", *argoCDmyInstalledHelmChart.Info)

	////////////////////////////////////////////////////////////////////////////////////////
	fmt.Println("Finished Deployment\n")
}

func connectToK8s() *kubernetes.Clientset {
	home, exists := os.LookupEnv("HOME")
	if !exists {
		home = "C:\\Users\\jcontent" // Change this later to more of a variable.
	}

	configPath := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		log.Panicln("failed to create K8s config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panicln("Failed to create K8s clientset")
	}

	return clientset
}

func connectToAks(ctx context.Context, subid string, rg string, aksname string) *armcontainerservice.ManagedClustersClientGetAccessProfileResponse {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}
	//ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory("AzureSubID", cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	res, err := clientFactory.NewManagedClustersClient().GetAccessProfile(ctx, "JCKubernetes-newprodjcvs2", "JCKubernetesCluster-newprodjcvs2", "clusterUser", nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	return &res
}

func Ptr[T any](v T) *T {
	return &v
}
