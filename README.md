***kubernetes-cpi***
==================
A CPI to deploy bosh releases to Kubernetes.

### Cloning and Building
------------------------

Clone this repo and build it. Using the following commands on a Linux or Mac OS X system:

```
$ pwd
/Users/{your-user-name}
$ mkdir -p src/github.ibm.com/Bluemix
$ cd src/github.ibm.com/Bluemix
$ git clone https://github.ibm.com:Bluemix/kubernetes-cpi.git
$ cd kubernetes-cpi
$ export GOPATH=/Users/{your-user-name}
$ ./bin/build
```
The executable output should now be located in: `out/cpi`.

### Running Tests
-----------------

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

 1. Integration Test will execute CPI actions against a real Kubernetes cluser, please setup CLI `bx` and `kubectl` described in [here].(https://github.ibm.com/Bluemix/kubernetes-cpi/issues/10)
 2. Provide API endpoint, credentials and cluster name etc. Edit `integration/env` and fill in all of them.
 
``` 
$ cat integration/env
 export BX_API=
 export BX_USERNAME=
 export BX_PASSWORD=
 export BX_ACCOUNTID=
 export CLUSTER_NAME=
 export SL_USERNAME=
 export SL_API_KEY=
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