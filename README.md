## node-comm-lib

This go library provide tools to produce k8s node communication matrix, i.e.  
a file that describes what ports the cluster listens to. 

We produce this matrix from the existing EndpointSlieces, and in order to fetch  
the relevant ones, the `endpointslices` package provide various querying methods. 
