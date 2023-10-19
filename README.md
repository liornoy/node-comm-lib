## node-comm-lib

This go library provide tools to produce k8s node communication matrix, i.e.  
a file that describes what ports the cluster listens to. 

We produce this matrix from the existing EndpointSlieces, and in order to fetch  
the relevant ones, the `endpointslices` package provide various querying methods. 


### e2etest:
To invoke the e2etest, start by exporting the "KUBECONFIG" variable, and then run 'make e2etest.' This test will generate two matrices:
One from the EndpointSlices when the host services are manually produced using the 'customEndpointSlices.json' file.
The other matrix is generated by running 'ss' on the nodes.
The test is expected to fail. You can find the output of the 'ss' command for each node and protocol,
as well as the raw communication matrices in the 'e2etest/artifacts' directory, and the diff will be printed as part of the test output.
