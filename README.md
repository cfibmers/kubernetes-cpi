***kubernetes-cpi***
==================
A CPI to deploy bosh releases to Kubernetes.

### Cloning and Building
------------------------

Clone this repo and build it. Using the following commands on a Linux or Mac OS X system:

```
$ MY_GO_PROJECTS=/path/to/go/projects
$ export GOPATH=$MY_GO_PROJECTS:$GOPATH
$ cd $MY_GO_PROJECTS
$ mkdir -p src/github.ibm.com/Bluemix
$ cd src/github.ibm.com/Bluemix
$ git clone git@github.ibm.com:Bluemix/kubernetes-cpi.git
$ cd kubernetes-cpi
$ ./bin/build
```
The executable output should now be located in: `out/cpi`.

### Running Tests
-----------------

#### - Install Ginkgo and Gomega

```shell
$ go get github.com/onsi/ginkgo/ginkgo
$ go get github.com/onsi/gomega
```

#### - Unit Tests

You can run the tests to make sure all is well, run unit tests with: `$ bin/test-unit` . The output of `$ bin/test-unit` should be similar to:

```
$ bin/test-unit

 Cleaning build artifacts...

 Formatting packages...

 Unit Testing packages:
[1497497714] Actions Suite - 114/115 specs - 7 nodes •••••••P••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••• SUCCESS! 595.328657ms
[1497497714] CPI Suite - 19/19 specs - 7 nodes ••••••••••••••••••• SUCCESS! 15.16537ms
[1497497714] Kubecluster Suite - 4/4 specs - 7 nodes •••• SUCCESS! 27.317174ms
[1497497714] Config Suite - 6/6 specs - 7 nodes •••••• SUCCESS! 14.76886ms
[1497497714] CPI Main Command Suite - 1/1 specs - 7 nodes • SUCCESS! 16.791359ms

Ginkgo ran 5 suites in 4.671339046s
Test Suite Passed

 Vetting packages for potential issues...

SWEET SUITE SUCCESS
```

#### - Integration Tests

 1. Integration Test will execute CPI actions against a real Kubernetes cluser, please setup CLI `bx` and `kubectl` described in [here](https://console.bluemix.net/docs/containers/cs_cli_install.html#cs_cli_install).
 2. Provide API endpoint, credentials and cluster name, etc., in your `.bashrc` or `.bash_profile`. There's an example config in [integration/env](integration/env).
 > **Note:**

 > - Please set $CLUSTER_NAME to the name of a functioning cluster, as integration tests will use the cluster directly to create pods, etc.
 > - Please keep the double quote in case there are some special characters of the values provided.



```
$ cat integration/env
export BX_API="{bluemix-api}"
export BX_USERNAME="{bluemix-user-name}"
export BX_PASSWORD="{bluemix-password}"
# Optionally supply BX_API_KEY (you must do this if your ID is federated). BX_USERNAME and BX_PASSWORD are not needed in that case, but you must supply one or the other.
# API keys can be associated with an account. If supplying an API key and it is not associated with an account, supply BX_ACCOUNTID.
# BX_ACCOUNTID must be supplied if using username and password.
export BX_ACCOUNTID="{bluemix-account}"
export CLUSTER_NAME="{existed-cluster}"
export SL_USERNAME="{softlayer-username}"
export SL_API_KEY="{softlayer-api-key}"

# example integration/env file
$ cat integration/env
export BX_API="api.ng.bluemix.net"
export BX_USERNAME="zhanggbj"
export BX_PASSWORD="password"
# export BX_API_KEY="abc"
export BX_ACCOUNTID="12345678910027ca24sdd12345678910"
export CLUSTER_NAME="cluster_integration"
export SL_USERNAME="zhanggbj"
export SL_API_KEY="12345678910570d12e2149b3fd12345678910499b34351a8db2512345678910"

$ source integration/env
```
3. Run integration tests with: `$ bin/test-integration` . The output of  `$ bin/test-unit` should be similar to:
```
$ bin/test-integration

 Cleaning build artifacts...

 Formatting packages...

 cd to base of project...

 Creating cpi binary...

 Integration Testing ALL CPI methods
Running Suite: Integration Suite
================================
Random Seed: 1497510180
Will run 0 of 0 specs


Ran 0 of 0 Specs in 0.000 seconds
SUCCESS! -- 0 Passed | 0 Failed | 0 Pending | 0 Skipped PASS

Running Suite: CreateVm Suite
=============================
Random Seed: 1497510180
Will run 2 of 2 specs

Integration test for create_vm create_vm in cluster
  returns error because empty parameters
  /Users/{your-user-name}/github.ibm.com/Bluemix/kubernetes-cpi/integration/create_vm/create_vm_test.go:63
•
------------------------------
Integration test for create_vm create_vm in cluster
  returns valid result because valid parameters
  /Users/{your-user-name}/src/github.ibm.com/Bluemix/kubernetes-cpi/integration/create_vm/create_vm_test.go:88

• [SLOW TEST:108.813 seconds]
Integration test for create_vm
/Users/{your-user-name}/src/github.ibm.com/Bluemix/kubernetes-cpi/integration/create_vm/create_vm_test.go:90
  create_vm in cluster
  /Users/{your-user-name}/src/github.ibm.com/Bluemix/kubernetes-cpi/integration/create_vm/create_vm_test.go:89
    returns valid result because valid parameters
    /Users/{your-user-name}/src/github.ibm.com/Bluemix/kubernetes-cpi/integration/create_vm/create_vm_test.go:88
------------------------------

Ran 2 of 2 Specs in 150.055 seconds
SUCCESS! -- 2 Passed | 0 Failed | 0 Pending | 0 Skipped PASS

Ginkgo ran 2 suites in 2m30.650380343s
Test Suite Passed

 Vetting packages for potential issues...

SWEET SUITE SUCCESS
```

### Managing dependencies
-------------------------

Dependencies are managed via [govendor](https://github.com/kardianos/govendor). See [vendor.json](vendor/vendor.json) for the current dependencies.
