# wavReader
Go version 1.21 or 1.17

go install go.k6.io/xk6/cmd/xk6@latest
export GOPATH=$HOME/go 
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
xk6 build --with github.com/RustTest/wavReader@latest
