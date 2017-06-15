kubernetes-cpi
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

 - Unit Tests

You can run the tests to make sure all is well, run unit tests with: `$ bin/test-unit` . The output should of `$ bin/test-unit` be similar to:

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
 - Integration Tests
(WIP)

### Managing dependencies
-------------------------

Dependencies are managed via [govendor](https://github.com/kardianos/govendor). See [vendor.json](vendor/vendor.json) for the current dependencies.