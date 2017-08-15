package config

import clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

type Cluster struct {
	Server                   string `json:"server"`
	InsecureSkipTLSVerify    bool   `json:"insecure_skip_tls_verify,omitempty"`
	CertificateAuthorityData string `json:"certificate_authority_data"`
}

type AuthInfo struct {
	ClientCertificateData string `json:"client_certificate_data,omitempty"`
	ClientKeyData         string `json:"client_key_data,omitempty"`
	Token                 string `json:"token,omitempty"`
	Username              string `json:"username,omitempty"`
	Password              string `json:"password,omitempty"`
	AuthProvider          string `json:"auth_provider,omitempty"`
	IdpIssuerURL          string `json:"idp_issuer_url,omitempty"`
	ClientSecret          string `json:"client_secret,omitempty"`
	ClientID              string `json:"client_id,omitempty"`
	RefreshToken          string `json:"refresh_token,omitempty"`
}

type Context struct {
	Cluster   string `json:"cluster"`
	AuthInfo  string `json:"user"`
	Namespace string `json:"namespace"`
}

type Kubernetes struct {
	Clusters       map[string]*Cluster  `json:"clusters"`
	AuthInfos      map[string]*AuthInfo `json:"users"`
	Contexts       map[string]*Context  `json:"contexts"`
	CurrentContext string               `json:"current_context"`
}

func (k Kubernetes) ClientConfig() clientcmdapi.Config {
	cc := clientcmdapi.NewConfig()
	cc.CurrentContext = k.CurrentContext
	for k, v := range k.Clusters {
		cc.Clusters[k] = v.api()
	}
	for k, v := range k.AuthInfos {
		cc.AuthInfos[k] = v.api()
	}
	for k, v := range k.Contexts {
		cc.Contexts[k] = v.api()
	}
	cc.Preferences = *clientcmdapi.NewPreferences()
	return *cc
}

func (a *AuthInfo) api() *clientcmdapi.AuthInfo {

	info := clientcmdapi.NewAuthInfo()
	info.Token = a.Token
	info.Username = a.Username
	info.Password = a.Password
	if len(a.ClientCertificateData) != 0 {
		info.ClientCertificateData = []byte(a.ClientCertificateData)
	}
	if len(a.ClientKeyData) != 0 {
		info.ClientKeyData = []byte(a.ClientKeyData)
	}

	if len(a.AuthProvider) > 0 {
		authProvider := &clientcmdapi.AuthProviderConfig{
			Name: a.AuthProvider,
		}

		if a.AuthProvider == "oidc" {
			authProvider.Config = make(map[string]string)
			authProvider.Config["idp-issuer-url"] = a.IdpIssuerURL
			authProvider.Config["client-secret"] = a.ClientSecret
			authProvider.Config["client-id"] = a.ClientID
			authProvider.Config["id-token"] = a.Token
			authProvider.Config["refresh-token"] = a.RefreshToken
		}

		info.AuthProvider = authProvider
	}
	return info
}

func (c *Cluster) api() *clientcmdapi.Cluster {
	cluster := clientcmdapi.NewCluster()
	cluster.Server = c.Server
	cluster.InsecureSkipTLSVerify = c.InsecureSkipTLSVerify
	if len(c.CertificateAuthorityData) != 0 {
		cluster.CertificateAuthorityData = []byte(c.CertificateAuthorityData)
	}
	return cluster
}

func (c *Context) api() *clientcmdapi.Context {
	context := clientcmdapi.NewContext()
	context.Cluster = c.Cluster
	context.AuthInfo = c.AuthInfo
	context.Namespace = c.Namespace
	return context
}
