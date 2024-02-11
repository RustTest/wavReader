# wavReader
Go version 1.21 or 1.17

go install go.k6.io/xk6/cmd/xk6@latest

export GOPATH=$HOME/go 

export PATH=$PATH:$GOROOT/bin:$GOPATH/bin

xk6 build --with github.com/RustTest/wavReader@latest

#run with 


PULSAR_ADDR=172.31.12.28:6650 ./k6 run --verbose voxflowTesting.js

If issues building the xk6 build --with github.com/RustTest/wavReader@latest 
try 
go clean -cache
go clean -modcache

